package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/404LifeFound/alertmanager-lark/internal/alert"
	larkgin "github.com/404LifeFound/lark-gin/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-lark/lark"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/rs/zerolog/log"
	kafka "github.com/segmentio/kafka-go"
)

type WebhookHandler struct {
	Writer *kafka.Writer
}

func (w *WebhookHandler) Webhook(c *gin.Context) {
	var webhook_event webhook.Message
	if err := c.ShouldBindBodyWithJSON(&webhook_event); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "webhook event is not valid",
			"error":   err.Error(),
		})
		return
	}

	kafka_msg, err := json.Marshal(webhook_event)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "json marshal webhook_event failed",
			"error":   err.Error(),
		})
		return
	}

	var key []byte
	if len(webhook_event.Alerts) > 0 {
		if fp := webhook_event.Alerts[0].Fingerprint; fp != "" {
			key = []byte(fp)
		}
	}

	retries := config.GlobalConfig.Kafka.WriteRetries
	if retries <= 0 {
		retries = 1
	}
	backoff := time.Duration(config.GlobalConfig.Kafka.WriteRetryBackoff) * time.Millisecond
	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		err = w.Writer.WriteMessages(ctx,
			kafka.Message{
				Key:   key,
				Value: kafka_msg,
			},
		)
		cancel()
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		if attempt < retries {
			time.Sleep(backoff)
		}
	}
	if lastErr != nil {
		c.Error(lastErr)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "write event to kafka failed",
			"error":   lastErr.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "received event and write to kafka successful",
		"error":   "",
	})
}

func RegisterHandlers(e *gin.Engine, w *kafka.Writer, bot *lark.Bot) error {
	e.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	webhook_handler := &WebhookHandler{
		Writer: w,
	}
	g := e.Group("/lark", nil)
	g.POST("/webhook", webhook_handler.Webhook)

	middleware := larkgin.NewLarkMiddleware()

	if config.GlobalConfig.Lark.EncryptKey != "" {
		middleware.WithEncryption(config.GlobalConfig.Lark.EncryptKey)
	}

	if config.GlobalConfig.Lark.VerificationToken != "" {
		middleware.WithTokenVerification(config.GlobalConfig.Lark.VerificationToken)
	}

	eventGroup := e.Group("/event")
	eventGroup.Use(middleware.LarkChallengeHandler(), middleware.LarkCardHandler())
	eventGroup.POST("/callback", func(c *gin.Context) {
		if card, ok := middleware.GetCardCallback(c); ok {
			log.Info().Msgf("received lark card callback: %+v", card)
			var action_value alert.CardActionValue

			err := json.Unmarshal(card.Event.Action.Value, &action_value)
			if err != nil {
				log.Error().Msgf("can't unmarshal card action value to CardActionValue: %v", err)
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"message": "invalid action request",
					"error":   err.Error(),
				})
				return
			}

			go func() {
				if action_value.Action != "resolve" {
					log.Info().Msgf("ignore card action: %s", action_value.Action)
					return
				}
				resolved := &alert.LarkCard{
					Title:        action_value.Title,
					Project:      action_value.Project,
					Time:         action_value.Time,
					GrafanaURL:   action_value.GrafanaURL,
					RunBookURL:   action_value.RunBookURL,
					Metric:       action_value.Metric,
					Description:  action_value.Description,
					AssignEmails: nil,
				}
				cardStr := resolved.NewResolvedCard()
				retries := config.GlobalConfig.Lark.SendRetries
				if retries <= 0 {
					retries = 1
				}
				backoff := time.Duration(config.GlobalConfig.Lark.SendRetryBackoff) * time.Millisecond
				var updateErr error
				var resp *lark.UpdateMessageResponse
				for attempt := 1; attempt <= retries; attempt++ {
					resp, updateErr = bot.UpdateMessage(card.Event.Context.OpenMessageID,
						lark.NewMsgBuffer(lark.MsgInteractive).
							Card(cardStr).
							Build(),
					)
					if updateErr == nil {
						if resp != nil && resp.Code != 0 {
							updateErr = fmt.Errorf("lark api error on update: code=%d, msg=%s", resp.Code, resp.Msg)
						}
					}
					if updateErr == nil {
						break
					}
					if attempt < retries {
						time.Sleep(backoff)
					}
				}
				if updateErr != nil {
					log.Error().Err(updateErr).Msgf("failed to update message %s", card.Event.Context.OpenMessageID)
				}
			}()
		} else {
			log.Warn().Msgf("no card callback parsed, headers: %+v", c.Request.Header)
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	return nil
}
