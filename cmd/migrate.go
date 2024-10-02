package cmd

import (
	"github.com/UNIwise/go-template/migrations"
	"github.com/UNIwise/go-template/pkg/connectors/database"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate database to latest migration",
	Long: `Migrate the database to the latest migration
in order to ensure that the database is up to date.`,
	Run: migrate,
}

// migrateCmd represents the migrate command.
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback last migration in database",
	Long:  `Rollback the last migration for maintenance purposes`,
	Run:   rollback,
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(rollbackCmd)
}

func initialize() (*gormigrate.Gormigrate, *logrus.Logger, error) {
	config, err := loadConfig("database", "log")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := GetLogger(config.Log)

	// Initialize database
	db, err := database.NewConnection(config.Database)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to initialize database connection")
	}

	m := gormigrate.New(db, &gormigrate.Options{}, []*gormigrate.Migration{
		migrations.Migration00001init,
	})

	return m, l, nil
}

func migrate(cmd *cobra.Command, args []string) {
	m, l, err := initialize()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize")
	}
	if err := m.Migrate(); err != nil {
		l.WithError(err).Fatal("Failed to migrate database")
	}
}

func rollback(cmd *cobra.Command, args []string) {
	m, l, err := initialize()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize")
	}
	if err := m.RollbackLast(); err != nil {
		l.WithError(err).Fatal("Failed to rollback database")
	}
}
