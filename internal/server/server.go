package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func NewGinEngine(lc fx.Lifecycle) *gin.Engine {
	e := gin.New()
	e.Use(
		logger.SetLogger(logger.WithLogger(func(_ *gin.Context, l zerolog.Logger) zerolog.Logger {
			return l.Output(gin.DefaultWriter).With().Logger()
		})),
		gin.Recovery(),
	)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.GlobalConfig.Http.Host, config.GlobalConfig.Http.Port),
		Handler: e,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					panic(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return e
}
