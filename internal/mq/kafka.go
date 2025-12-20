package mq

import (
	"context"

	"github.com/404LifeFound/alertmanager-lark/config"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/fx"
)

func NewKafkaReader(lc fx.Lifecycle) *kafka.Reader {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  config.GlobalConfig.Kafka.Brokers,
		Topic:    config.GlobalConfig.Kafka.Topic,
		GroupID:  config.GlobalConfig.Kafka.ConsumerGroup,
		MaxBytes: config.GlobalConfig.Kafka.MaxBytes,
	})

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return r.Close()
		},
	})

	return r
}

func NewKafkaWriter(lc fx.Lifecycle) *kafka.Writer {
	w := &kafka.Writer{
		Addr:  kafka.TCP(config.GlobalConfig.Kafka.Brokers...),
		Topic: config.GlobalConfig.Kafka.Topic,
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return w.Close()
		},
	})

	return w
}
