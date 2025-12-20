package worker

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/404LifeFound/alertmanager-lark/internal/alert"
	"github.com/chyroc/lark"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/rs/zerolog/log"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/fx"
)

func Run(lc fx.Lifecycle, reader *kafka.Reader, lark *lark.Lark) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				for {
					m, err := reader.ReadMessage(ctx)
					if err != nil {
						break
					}
					log.Info().Msgf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))

					var webhook_event webhook.Message
					if err = json.Unmarshal(m.Value, &webhook_event); err != nil {
						log.Error().Err(err).Msgf("failed to unmarshal %v", string(m.Value))
					}

					for _, a := range webhook_event.Alerts {
						c := &alert.LarkCard{
							Title:        a.Labels["alertname"],
							Project:      a.Labels["Project"],
							Time:         a.StartsAt.String(),
							AssignEmails: strings.Split(a.Labels["NotifyEmails"], ","),
							GrafanaURL:   a.Labels["GrafanaURL"],
							RunBookURL:   a.Labels["RunBookURL"],
							Metric:       a.GeneratorURL,
							Description:  a.Annotations["description"],
						}
						card_s := c.NewLarkCard().String()
						log.Info().Msgf("card string is: %s", card_s)
						_, _, err = lark.Message.Send().ToChatID(config.GlobalConfig.Lark.ChatID).SendCard(ctx, card_s)
						if err != nil {
							log.Error().Err(err).Msgf("faild to send card message %v to chat: %v", string(m.Value), config.GlobalConfig.Lark.ChatID)
						}

					}

				}
			}()
			return nil
		},
	})
}
