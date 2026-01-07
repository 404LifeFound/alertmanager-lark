package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

func NewGinEngine(lc fx.Lifecycle) *gin.Engine {
	e := gin.New()
	e.Use(
		logger.SetLogger(logger.WithLogger(func(_ *gin.Context, l zerolog.Logger) zerolog.Logger {
			return l.Output(gin.DefaultWriter).With().Logger()
		}), logger.WithSkipPath([]string{"/healthz"})), // We can now safely log /lark/callback
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

func bodyLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/event/callback" && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				log.Error().Err(err).Msg("failed to read request body for buffering")
				c.Next()
				return
			}
			log.Info().
				Int("buffered_body_size", len(bodyBytes)).
				Str("buffered_body_content", string(bodyBytes)).
				Msg("request body buffered")
			if config.GlobalConfig.Lark.VerificationToken != "" {
				nonce := c.Request.Header.Get("X-Lark-Request-Nonce")
				timestamp := c.Request.Header.Get("X-Lark-Request-Timestamp")
				signature := c.Request.Header.Get("X-Lark-Signature")
				mac := hmac.New(sha256.New, []byte(config.GlobalConfig.Lark.VerificationToken))
				mac.Write([]byte(timestamp))
				mac.Write([]byte(nonce))
				mac.Write(bodyBytes)
				expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
				abbr := func(s string) string {
					if len(s) > 8 {
						return s[:8] + "..."
					}
					return s
				}
				log.Info().
					Str("req_sig_prefix", abbr(signature)).
					Str("exp_sig_prefix", abbr(expected)).
					Bool("sig_matched", expected == signature).
					Msg("token verification precheck for card callback")
			}
			c.Request.Body.Close()
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Request.ContentLength = int64(len(bodyBytes))
		}
		c.Next()
	}
}
