package server

import (
	"context"
	"net/http"
	"time"

	"github.com/callmemars1/ytdlp-http/internal/configurations"
	"github.com/callmemars1/ytdlp-http/internal/server/middlewares"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func RunHTTPServer(
	lc fx.Lifecycle,
	config *configurations.Config,
	authMiddleware *middlewares.AuthMiddleware,
	handlers []Handler,
	logger *zap.Logger,
) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	router.Use(gin.Recovery())
	router.Use(ginLogger(logger))
	router.Use(authMiddleware.Authenticate())

	for _, handler := range handlers {
		handler.SetupRoute(router)
	}

	srv := &http.Server{
		Addr:    config.Server.Addr,
		Handler: router,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting HTTP server", zap.String("addr", config.Server.Addr))
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("Server failed to start", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down HTTP server")
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	})
	return srv
}

func ginLogger(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Info("HTTP request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}
