package database

import (
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"
	"github.com/UNIwise/go-template/pkg/health"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	healthExecutionInterval = time.Second * 10
	connMaxIdleTime         = time.Minute
	connMaxLifetime         = time.Minute * 10
)

type Config struct {
	DSN   string `validate:"required"`
	Debug bool   `default:"false"`
}

func NewConnection(c Config) (*gorm.DB, error) {
	logMode := logger.Silent
	if c.Debug {
		logMode = logger.Info
	}
	db, err := gorm.Open(mysql.Open(c.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logMode),
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open database connection")
	}

	odb, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get underlying database connection")
	}

	odb.SetConnMaxIdleTime(connMaxIdleTime)
	odb.SetConnMaxLifetime(connMaxLifetime)

	return db, nil
}

func RegisterChecks(gdb *gorm.DB, h health.Server) error {
	db, err := gdb.DB()
	if err != nil {
		return errors.Wrap(err, "failed to get database connection")
	}

	c, err := checks.NewPingCheck("database", db)
	if err != nil {
		return errors.Wrap(err, "Failed to instantiate database healthcheck")
	}

	if err := h.RegisterCheck(c, gosundheit.ExecutionPeriod(healthExecutionInterval)); err != nil {
		return errors.Wrap(err, "Failed to register database healthcheck")
	}

	return nil
}
