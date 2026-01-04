package server

import (
	"context"
	"encoding/json"
	"net/http"

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

	if err := w.Writer.WriteMessages(context.Background(),
		kafka.Message{
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
