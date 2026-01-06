package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/chyroc/lark"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/alertmanager/notify/webhook"
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

func RegisterHandlers(e *gin.Engine, w *kafka.Writer, l *lark.Lark) error {
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
	g.POST("/callback", func(c *gin.Context) {
		l.EventCallback.ListenCallback(c.Request.Context(), c.Request.Body, c.Writer)
	})
	return nil
}
