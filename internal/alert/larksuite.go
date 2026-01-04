package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/chyroc/lark"
	"github.com/chyroc/lark/card"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

func NewLark(lc fx.Lifecycle) *lark.Lark {
	l := lark.New(
		lark.WithAppCredential(config.GlobalConfig.Lark.AppID, config.GlobalConfig.Lark.AppSecret),
		lark.WithEventCallbackVerify(config.GlobalConfig.Lark.EncryptKey, config.GlobalConfig.Lark.VerificationToken),
		lark.WithNonBlockingCallback(true),
		lark.WithOpenBaseURL("https://open.larksuite.com"),
		lark.WithWWWBaseURL("https://www.larksuite.com"),
	)

	return l
}

type LarkCard struct {
	Title        string
	Project      string
	Time         string
	GrafanaURL   string
	RunBookURL   string
	AssignEmails []string
	Metric       string
	Description  string
}

type CardActionValue struct {
	Title       string `json:"title"`
	Project     string `json:"project"`
	Time        string `json:"time"`
	GrafanaURL  string `json:"grafana_url"`
	RunBookURL  string `json:"runbook_url"`
	Metric      string `json:"metric"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

func (l *LarkCard) ParseTime() (string, error) {
	t, err := time.Parse(time.RFC3339Nano, l.Time)
	if err != nil {
		t, err = time.Parse(time.RFC3339, l.Time)
		if err != nil {
			log.Error().Err(err).Msgf("failed to parse time: %s", l.Time)
			return "", err
		}
	}
	return t.Format("2006-01-02 15:04:05.000"), nil

}

// http://example.com:9090/graph?g0.expr=go_memstats_alloc_bytes+%3E+0\u0026g0.tab=1
func (l *LarkCard) ParseExpr() (string, error) {
	u, err := url.Parse(l.Metric)
	if err != nil {
		log.Error().Err(err).Msgf("faild to parse url of metric: %v", l.Metric)
		return "", err
	}

	log.Info().Msgf("parsed url: %v", u)

	expr := u.Query().Get("g0.expr")
	exprDecoded, err := url.QueryUnescape(expr)
	if err != nil {
		log.Error().Err(err).Msgf("faild to decode metric: %v", expr)
		return "", err
	}

	log.Info().Msgf("parsed metric: %s", exprDecoded)

	return exprDecoded, nil
}

func (l *LarkCard) WithMetric(metric string) *LarkCard {
	l.Metric = metric
	return l
}

func (l *LarkCard) WithTime(t string) *LarkCard {
	l.Time = t
	return l
}

func (l *LarkCard) SetTitle() string {
	return fmt.Sprintf("🚨 %s", l.Title)
}

func (l *LarkCard) ProjectMD() string {
	return fmt.Sprintf("**📦 Project:**\n%s", l.Project)
}

func (l *LarkCard) TimeMD() string {
	return fmt.Sprintf("**🕐 Time:**\n%s", l.Time)
}

func (l *LarkCard) GrafanaURLMD() string {
	return fmt.Sprintf("**🔗 Grafana: **\n%s", l.GrafanaURL)
}

func (l *LarkCard) RunbookMD() string {
	return fmt.Sprintf("**📘 Runbook: **\n%s", l.RunBookURL)
}

func (l *LarkCard) AssignEmailMD() string {
	assign_s := "**👤 Assigned to: **\n"
	filtered := make([]string, 0, len(l.AssignEmails))
	for _, e := range l.AssignEmails {
		e = strings.TrimSpace(e)
		if e != "" {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		assign_s += "<at id=all></at>"
		return assign_s
	}
	for _, e := range filtered {
		assign_s += fmt.Sprintf("<at email=%s></at>", e)
	}
	return assign_s
}

func (l *LarkCard) MetricMD() string {
	return fmt.Sprintf("**📊 Metric: **\n```\n%s\n```", l.Metric)
}

func (l *LarkCard) DescriptionMD() string {
	return fmt.Sprintf("**👉 Description: **\n%s", l.Description)
}

func (l *LarkCard) NewLarkCard() *lark.MessageContentCard {
	metric, err := l.ParseExpr()
	if err != nil {
		log.Error().Err(err).Send()
	} else {
		l.WithMetric(metric)
	}

	t, err := l.ParseTime()
	if err != nil {
		log.Error().Err(err).Send()
	} else {
		l.WithTime(t)
	}

	c := card.Card(
		card.ColumnSet(
			card.Column(
				card.Markdown(l.ProjectMD()),
			).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			card.Column(
				card.Markdown(l.TimeMD()),
			).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
		),
		card.ColumnSet(
			card.Column(
				card.Markdown(l.GrafanaURLMD()),
			).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			card.Column(
				card.Markdown(l.RunbookMD()),
			).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
		),
		card.ColumnSet(
			card.Column(
				card.Markdown(l.AssignEmailMD()),
			).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
		),
		card.ColumnSet(
			card.Column(
				card.Markdown(l.MetricMD()),
			).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
		),
		card.ColumnSet(
			card.Column(
				card.Markdown(l.DescriptionMD()),
			).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
		),
		card.Action(
			card.Button("Resolved", nil).SetPrimary().SetValue(CardActionValue{
				Title:       l.Title,
				Project:     l.Project,
				Time:        l.Time,
				GrafanaURL:  l.GrafanaURL,
				RunBookURL:  l.RunBookURL,
				Metric:      l.Metric,
				Description: l.Description,
				Action:      "resolve",
			}),
		),
	).SetHeader(
		card.Header(l.SetTitle()).SetRed(),
	)

	return c
}

func CardEventCallback(l *lark.Lark) {
	l.EventCallback.HandlerEventCard(func(ctx context.Context, cli *lark.Lark, event *lark.EventCardCallback) (string, error) {
		if event == nil || event.Action == nil {
			return "", nil
		}
		var val CardActionValue
		if err := json.Unmarshal(event.Action.Value, &val); err != nil {
			return "", nil
		}
		if val.Action != "resolve" {
			return "", nil
		}
		lc := &LarkCard{
			Title:       val.Title,
			Project:     val.Project,
			Time:        val.Time,
			GrafanaURL:  val.GrafanaURL,
			RunBookURL:  val.RunBookURL,
			Metric:      val.Metric,
			Description: val.Description,
		}
		metric, err := lc.ParseExpr()
		if err == nil {
			lc.WithMetric(metric)
		}
		t, err := lc.ParseTime()
		if err == nil {
			lc.WithTime(t)
		}
		c := card.Card(
			card.ColumnSet(
				card.Column(
					card.Markdown(lc.ProjectMD()),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
				card.Column(
					card.Markdown(lc.TimeMD()),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			),
			card.ColumnSet(
				card.Column(
					card.Markdown(lc.GrafanaURLMD()),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
				card.Column(
					card.Markdown(lc.RunbookMD()),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			),
			card.ColumnSet(
				card.Column(
					card.Markdown(lc.AssignEmailMD()),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			),
			card.ColumnSet(
				card.Column(
					card.Markdown("**状态：**\n✅ Resolved"),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			),
			card.ColumnSet(
				card.Column(
					card.Markdown(lc.MetricMD()),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			),
			card.ColumnSet(
				card.Column(
					card.Markdown(lc.DescriptionMD()),
				).SetWidth("weighted").SetWeight(1).SetTopVerticalAlign(),
			),
		).SetHeader(
			card.Header(fmt.Sprintf("✅ %s", lc.Title)).SetGreen(),
		)
		return c.String(), nil
	})
}
