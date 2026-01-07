package alert

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/go-lark/lark"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

func NewLark(lc fx.Lifecycle) *lark.Bot {
	bot := lark.NewChatBot(config.GlobalConfig.Lark.AppID, config.GlobalConfig.Lark.AppSecret)
	bot.SetDomain(lark.DomainLark)
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			bot.StartHeartbeat()
			return nil
		},
		OnStop: func(context.Context) error {
			bot.StopHeartbeat()
			return nil
		},
	})
	return bot
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

func (l *LarkCard) SetResolvedTitle() string {
	return fmt.Sprintf("%s (Resolved)", l.SetTitle())
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

func (l *LarkCard) NewLarkCard() string {
	metric, err := l.ParseExpr()
	if err == nil && metric != "" {
		l.WithMetric(metric)
	}
	t, err := l.ParseTime()
	if err == nil && t != "" {
		l.WithTime(t)
	}
	b := lark.NewCardBuilder()
	c := b.Card(
		b.ColumnSet(
			b.Column(b.Markdown(l.ProjectMD())).Width("weighted").Weight(1),
			b.Column(b.Markdown(l.TimeMD())).Width("weighted").Weight(1),
		).FlexMode("none"),
		b.ColumnSet(
			b.Column(b.Markdown(l.GrafanaURLMD())).Width("weighted").Weight(1),
			b.Column(b.Markdown(l.RunbookMD())).Width("weighted").Weight(1),
		).FlexMode("none"),
		b.Markdown(l.AssignEmailMD()),
		b.Markdown(l.MetricMD()),
		b.Markdown(l.DescriptionMD()),
		b.Action(
			b.Button(b.Text("Resolved")).Primary().Value(map[string]interface{}{
				"title":       l.Title,
				"project":     l.Project,
				"time":        l.Time,
				"grafana_url": l.GrafanaURL,
				"runbook_url": l.RunBookURL,
				"metric":      l.Metric,
				"description": l.Description,
				"action":      "resolve",
			}),
		),
	).Title(l.SetTitle()).Red().UpdateMulti(true)
	return c.String()
}

func (l *LarkCard) NewResolvedCard() string {
	b := lark.NewCardBuilder()
	c := b.Card(
		b.ColumnSet(
			b.Column(b.Markdown(l.ProjectMD())).Width("weighted").Weight(1),
			b.Column(b.Markdown(l.TimeMD())).Width("weighted").Weight(1),
		).FlexMode("none"),
		b.ColumnSet(
			b.Column(b.Markdown(l.GrafanaURLMD())).Width("weighted").Weight(1),
			b.Column(b.Markdown(l.RunbookMD())).Width("weighted").Weight(1),
		).FlexMode("none"),
		b.Markdown(l.AssignEmailMD()),
		b.Markdown(l.MetricMD()),
		b.Markdown(l.DescriptionMD()),
	).Title(l.SetResolvedTitle()).Green().UpdateMulti(true)
	return c.String()
}
