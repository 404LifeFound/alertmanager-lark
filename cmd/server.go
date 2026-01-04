/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/404LifeFound/alertmanager-lark/internal/alert"
	"github.com/404LifeFound/alertmanager-lark/internal/mq"
	"github.com/404LifeFound/alertmanager-lark/internal/server"
	"github.com/404LifeFound/alertmanager-lark/internal/worker"
	"github.com/ipfans/fxlogger"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func NewServerCmd() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start http server",
		Long:  "Start a http server for webhook",
		Run: func(cmd *cobra.Command, args []string) {
			app := fx.New(
				fx.Provide(
					server.NewGinEngine,
					mq.NewKafkaReader,
					mq.NewKafkaWriter,
					alert.NewLark,
				),
				fx.Invoke(
					alert.CardEventCallback,
					server.RegisterHandlers,
					worker.Run,
				),
				fx.WithLogger(fxlogger.WithZerolog(log.Logger)),
			)
			app.Run()
		},
	}

	flags := serverCmd.Flags()
	installFlags(flags)
	return serverCmd
}

func installFlags(flags *pflag.FlagSet) {
	log.Debug().Msg("start to install server flags")
	flags.StringP("http-host", "H", "0.0.0.0", "http host")
	viper.BindPFlag("http.host", flags.Lookup("http-host"))

	flags.IntP("http-port", "P", 8080, "http port")
	viper.BindPFlag("http.port", flags.Lookup("http-port"))

	flags.StringSlice("kafka-brokers", []string{"localhost:9092"}, "kafka brokers list")
	viper.BindPFlag("kafka.brokers", flags.Lookup("kafka-brokers"))

	flags.String("kafka-topic", "webhook-topic", "kafka topic")
	viper.BindPFlag("kafka.topic", flags.Lookup("kafka-topic"))

	flags.String("kafka-consumer-group", "webhook-consumer", "kafka consumer group ID")
	viper.BindPFlag("kafka.consumerGroup", flags.Lookup("kafka-consumer-group"))

	flags.Int("kafka-max-bytes", 10e6, "Max size of kafka message produce or consume(bytes)")
	viper.BindPFlag("kafka.maxBytes", flags.Lookup("kafka-max-bytes"))

	flags.String("lark-app-id", "", "lark appID")
	viper.BindPFlag("lark.appID", flags.Lookup("lark-app-id"))

	flags.String("lark-app-secret", "", "lark appSecret")
	viper.BindPFlag("lark.appSecret", flags.Lookup("lark-app-secret"))

	flags.String("lark-chat-id", "", "lark chatID")
	viper.BindPFlag("lark.chatID", flags.Lookup("lark-chat-id"))

	flags.String("lark-encrypt-key", "", "lark callback encrypt key")
	viper.BindPFlag("lark.encryptKey", flags.Lookup("lark-encrypt-key"))

	flags.String("lark-verification-token", "", "lark callback veritication token")
	viper.BindPFlag("lark.verificationToken", flags.Lookup("lark-verification-token"))
}
