package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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

	// 使用 Fingerprint 作为消息 key，利于同告警分区与幂等
	var key []byte
	if len(webhook_event.Alerts) > 0 {
		if fp := webhook_event.Alerts[0].Fingerprint; fp != "" {
			key = []byte(fp)
		}
	}

	// 写入 Kafka 设置超时，避免阻塞请求
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	if err := w.Writer.WriteMessages(ctx,
		kafka.Message{
			Key:   key,
			Value: kafka_msg,
		},
	); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "write event to kafka failed",
			"error":   err.Error(),
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
