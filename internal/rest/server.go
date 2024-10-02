package rest

import (
	"context"
	"fmt"

	"github.com/UNIwise/go-template/internal/authorization"
	"github.com/UNIwise/go-template/internal/rest/controllers"
	"github.com/UNIwise/go-template/internal/rest/helpers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	gzipCompressionLevel = 5
)

type Config struct {
	Port uint32 `validate:"required"`
}

type Server struct {
	echo   *echo.Echo
	config Config
}

func NewServer(conf Config, logger *logrus.Entry, authorizationService authorization.Service) (*Server, error) {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	e.Validator = helpers.NewValidator()
	e.HTTPErrorHandler = helpers.HTTPErrorHandler

	// prometheusMiddleware, err := xmid.Prometheus()
	// if err != nil {
	// 	return nil, errors.Wrap(err, "Failed to create prometheus middleware")
	// }

	e.Use(
		middleware.Recover(),
		// prometheusMiddleware,
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"*"},
		}),
		// xmid.RequestIDWithConfig(xmid.RequestIDConfig{
		// 	Generator: middleware.DefaultRequestIDConfig.Generator,
		// 	Header:    "X-Trace-ID",
		// 	Skipper:   middleware.DefaultRequestIDConfig.Skipper,
		// }),
		// middleware.GzipWithConfig(middleware.GzipConfig{
		// 	Level:   gzipCompressionLevel,
		// 	Skipper: middleware.DefaultGzipConfig.Skipper,
		// }),
	)

	root := e.Group("/hackathon")

	controllers.Register(root, logger, authorizationService)

	return &Server{
		echo:   e,
		config: conf,
	}, nil
}

func (s *Server) Start() error {
	return errors.Wrap(s.echo.Start(fmt.Sprintf(":%d", s.config.Port)), "Failed to start server")
}

func (s *Server) Shutdown(ctx context.Context) error {
	return errors.Wrap(s.echo.Shutdown(ctx), "Failed to shutdown server")
}
