package worker

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/404LifeFound/alertmanager-lark/internal/alert"
	"github.com/chyroc/lark"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/rs/zerolog/log"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/fx"
)

func Run(lc fx.Lifecycle, reader *kafka.Reader, lark *lark.Lark) {
	workerCtx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				backoff := time.Second
				for {
					m, err := reader.ReadMessage(workerCtx)
					if err != nil {
						if workerCtx.Err() != nil {
							// lifecycle cancelled
							log.Info().Msg("worker context cancelled, exiting kafka consumer")
							break
						}
						log.Error().Err(err).Msg("read message failed, will retry")
						time.Sleep(backoff)
						if backoff < 30*time.Second {
							backoff *= 2
						}
						continue
					}
					// reset backoff on success
					backoff = time.Second
					log.Info().Msgf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))

					var webhook_event webhook.Message
					if err = json.Unmarshal(m.Value, &webhook_event); err != nil {
						log.Error().Err(err).Msgf("failed to unmarshal %v", string(m.Value))
					}

					for _, a := range webhook_event.Alerts {
						alertname := FindFirstValue(a, "N/A", "alertname")
						project := FindFirstValue(a, "N/A", "Project", "project")
						notify_emails := FindFirstValue(a, "", "NotifyEmails", "notify_emails")
						grafana_url := FindFirstValue(a, "N/A", "GrafanaURL", "grafana_url")
						runbook_url := FindFirstValue(a, "N/A", "RunBookURL", "runbook_url")
						description := FindFirstValue(a, "N/A", "description", "summary")

						c := &alert.LarkCard{
							Title:        alertname,
							Project:      project,
							Time:         a.StartsAt.Format(time.RFC3339Nano),
							AssignEmails: strings.Split(notify_emails, ","),
							GrafanaURL:   grafana_url,
							RunBookURL:   runbook_url,
							Metric:       a.GeneratorURL,
							Description:  description,
						}
						card_s := c.NewLarkCard().String()
						log.Info().Msgf("card string is: %s", card_s)
						retries := config.GlobalConfig.Lark.SendRetries
						if retries <= 0 {
							retries = 1
						}
						backoff := time.Duration(config.GlobalConfig.Lark.SendRetryBackoff) * time.Millisecond
						var sendErr error
						for attempt := 1; attempt <= retries; attempt++ {
							_, _, sendErr = lark.Message.Send().ToChatID(config.GlobalConfig.Lark.ChatID).SendCard(workerCtx, card_s)
							if sendErr == nil {
								break
							}
							if attempt < retries {
								time.Sleep(backoff)
							}
						}
						if sendErr != nil {
							log.Error().Err(sendErr).Msgf("faild to send card message %v to chat: %v", string(m.Value), config.GlobalConfig.Lark.ChatID)
						}

					}

				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("stopping kafka consumer")
			cancel()
			return nil
		},
	})
}
