package alert

import (
	"fmt"
	"net/url"
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

func (l *LarkCard) ParseTime() (string, error) {
	str := "2018-08-03 09:52:26.739266876 +0200 +0200"
	layout := "2006-01-02 15:04:05.999999999 +0200 +0200"

	t, err := time.Parse(layout, str)
	if err != nil {
		panic(err)
	}

	t, err = time.Parse(layout, l.Time)
	if err != nil {
		log.Error().Err(err).Msgf("failed to parse time: %s", l.Time)
		return "", err
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
	if len(l.AssignEmails) == 0 {
		assign_s += "<at id=all></at>"
		return assign_s
	}

	for _, e := range l.AssignEmails {
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
			card.Button("Resolved", nil).SetPrimary().SetValue("resolve test value"),
		),
	).SetHeader(
		card.Header(l.SetTitle()).SetRed(),
	)

	return c
}
