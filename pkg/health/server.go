package health

import (
	"context"
	"fmt"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Port    int  `default:"8081"`
	Enabled bool `default:"false"`
}

type Server interface {
	Start() (err error)
	Shutdown(ctx context.Context) (err error)
	RegisterCheck(check gosundheit.Check, opts ...gosundheit.CheckOption) error
}

type ServerImpl struct {
	echo     *echo.Echo
	sundheit gosundheit.Health
	config   Config
}

func NewServer(config Config) (*ServerImpl, error) {
	e := echo.New()

	h := gosundheit.New()

	e.HideBanner = true
	e.HidePort = true

	e.GET("/healthz", echo.WrapHandler(healthhttp.HandleHealthJSON(h)))

	return &ServerImpl{
		echo:     e,
		sundheit: h,
		config:   config,
	}, nil
}

func (s *ServerImpl) Start() error {
	if !s.config.Enabled {
		logrus.Info("Health check disabled")

		return nil
	}

	return s.echo.Start(fmt.Sprintf(":%d", s.config.Port))
}

func (s *ServerImpl) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

func (s *ServerImpl) RegisterCheck(check gosundheit.Check, opts ...gosundheit.CheckOption) error {
	return s.sundheit.RegisterCheck(check, opts...)
}
