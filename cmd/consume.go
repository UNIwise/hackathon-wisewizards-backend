package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/UNIwise/go-template/pkg/connectors/database"
	"github.com/UNIwise/go-template/pkg/connectors/watermill/nats"
	"github.com/UNIwise/go-template/pkg/health"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// consumeCmd represents the consume command.
var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "Start a nats consumer",
	Long: `Start a nats consumer, which will consume messages 
from a nats-streaming cluster and create flows in the database.
`,
	Run: consume,
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(consumeCmd)
}

func consume(cmd *cobra.Command, args []string) {
	config, err := loadConfig("log", "database", "nats", "rabbitmq", "health", "authorization")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := GetLogger(config.Log)
	l.Info("Initializing server")

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

	// authorizationService := authorization.NewService(config.Authorization)

	// Initialize watermill connectors
	natsConnector, err := nats.NewConnector(config.NATS, l.WithField("subsystem", "NATS connector"))
	if err != nil {
		l.WithError(err).Fatal("Failed to create NATS connector")
	}

	// Initialize the workers
	// _, err = natsWorker.NewWorker(config.NATSWorker, natsConnector, authorizationService, flowService, l.WithField("subsystem", "NATS worker"))
	// if err != nil {
	// 	l.WithError(err).Fatal("Failed to create NATS worker")
	// }

	// _, err = rabbitMQWorker.NewWorker(config.RabbitMQWorker, rabbitMQConnector, flowService, l.WithField("subsystem", "RabbitMQ worker"))
	// if err != nil {
	// 	l.WithError(err).Fatal("Failed to create RabbitMQ worker")
	// }

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start the consumers
	l.Info("starting NATS consumer")
	go func() {
		if err := natsConnector.Start(ctx); err != nil {
			l.WithError(err).Error("Failed to start NATS worker")
			cancel()
		}
	}()

	// Start the health server
	l.WithField("port", config.Health.Port).Info("Health server starting")
	go func() {
		if err := healthServer.Start(); err != nil {
			l.WithError(err).Error("Failed to start health server")
			cancel()
		}
	}()

	// Wait for the context to be cancelled
	<-ctx.Done()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), shutdownPeriod)
	defer stopCancel()

	// Stop the consumers
	l.Info("Stopping NATS consumer")
	if err := natsConnector.Stop(ctx); err != nil {
		l.WithError(err).Error("Failed to stop NATS worker")
	}

	// Stop the health server
	l.Info("Stopping health server")
	if err := healthServer.Shutdown(stopCtx); err != nil {
		l.WithError(err).Error("Failed to stop health server")
	}
}
