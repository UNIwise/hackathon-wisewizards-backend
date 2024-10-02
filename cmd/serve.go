package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/UNIwise/go-template/internal/authorization"
	"github.com/UNIwise/go-template/internal/rest"
	"github.com/UNIwise/go-template/pkg/connectors/database"
	"github.com/UNIwise/go-template/pkg/health"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the http server",
	Long: `Start the example http server
Which is a simple rest api for flows.`,
	Run: serve,
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(serveCmd)
}

func serve(cmd *cobra.Command, args []string) {
	config, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := GetLogger(config.Log)

	healthServer, err := health.NewServer(config.Health)
	if err != nil {
		l.WithError(err).Fatal("Failed to create health server")
	}

	// Initialize database
	db, err := database.NewConnection(config.Database)
	if err != nil {
		l.WithError(err).Fatal("Failed to create database connection")
	}

	if err := database.RegisterChecks(db, healthServer); err != nil {
		l.WithError(err).Fatal("Failed to register database checks")
	}

	authorizationService := authorization.NewService(config.Authorization)

	// Initialize rest server
	httpServer, err := rest.NewServer(config.HTTP, l.WithField("subsystem", "http"), authorizationService)
	if err != nil {
		l.WithError(err).Fatal("Failed to create http server")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	l.WithField("port", config.Health.Port).Info("Health server starting")
	go func() {
		if err := healthServer.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			l.WithError(err).Fatal("Failed to start health server")
		}
	}()

	// Start the servers
	l.WithField("port", config.HTTP.Port).Info("REST server starting")
	go func() {
		if err := httpServer.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			l.WithError(err).Error("Failed to start http server")
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the servers
	l.Info("Shutting down")

	shutdownctx, stop := context.WithTimeout(context.Background(), shutdownPeriod)
	defer stop()

	if err := httpServer.Shutdown(shutdownctx); err != nil {
		l.WithError(err).Error("Failed to shutdown http server")
	}

	if err := healthServer.Shutdown(shutdownctx); err != nil {
		l.WithError(err).Error("Failed to shutdown health server")
	}
}
