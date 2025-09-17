package main

import (
	"github.com/callmemars1/ytdlp-http/internal/configurations"
	"github.com/callmemars1/ytdlp-http/internal/s3"
	"github.com/callmemars1/ytdlp-http/internal/server"
	"github.com/callmemars1/ytdlp-http/internal/server/handlers"
	"github.com/callmemars1/ytdlp-http/internal/server/middlewares"
	"github.com/callmemars1/ytdlp-http/internal/utils"
	"github.com/callmemars1/ytdlp-http/internal/ytdlp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(
			provideConfig,
			provideLogger,
			provideYtdlpService,
			provideS3Service,
			provideAuthMiddleware,
			AsHandler(handlers.NewDownloadHandler),
			AsHandler(handlers.NewUploadHandler),
		),

		fx.Invoke(
			fx.Annotate(
				server.RunHTTPServer,
				fx.ParamTags(``, ``, ``, `group:"handlers"`, ``),
			),
		),
	).Run()
}

func provideConfig() (*configurations.Config, error) {
	return configurations.NewConfig()
}

func provideLogger() (*zap.Logger, error) {
	return utils.NewLogger()
}

func provideYtdlpService(logger *zap.Logger) *ytdlp.Service {
	return ytdlp.NewService(logger)
}

func provideS3Service(config *configurations.Config, logger *zap.Logger) (*s3.Service, error) {
	return s3.NewService(&config.S3, logger)
}

func provideAuthMiddleware(config *configurations.Config, logger *zap.Logger) *middlewares.AuthMiddleware {
	return middlewares.NewAuthMiddleware(&config.Auth, logger)
}

func AsHandler(h any) any {
	return fx.Annotate(
		h,
		fx.As(new(server.Handler)),
		fx.ResultTags(`group:"handlers"`),
	)
}
