package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/UNIwise/go-template/internal/authorization"
	"github.com/UNIwise/go-template/internal/rest"
	"github.com/UNIwise/go-template/pkg/connectors/database"
	"github.com/UNIwise/go-template/pkg/connectors/watermill/nats"
	"github.com/UNIwise/go-template/pkg/connectors/watermill/rabbitmq"
	"github.com/UNIwise/go-template/pkg/health"
	"github.com/go-playground/validator"
	"github.com/mcuadros/go-defaults"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

const (
	slowQueryThreshold = 5 * time.Second
	shutdownPeriod     = 15 * time.Second
)

type Config struct {
	Log           LogConfig            `mapstructure:"log"`
	Database      database.Config      `mapstructure:"database"`
	HTTP          rest.Config          `mapstructure:"http"`
	NATS          nats.Config          `mapstructure:"nats"`
	RabbitMQ      rabbitmq.Config      `mapstructure:"rabbitmq"`
	Health        health.Config        `mapstructure:"health"`
	Authorization authorization.Config `mapstructure:"authorization"`
}

type LogConfig struct {
	Level  string `mapstructure:"level" default:"info" validate:"oneof=debug info warn error fatal panic"`
	Format string `mapstructure:"format" default:"text" validate:"oneof=json text"`
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "go-template",
	Short: "A template go application",
	Long: `This is a template go application
with examples for an http server, a grpc server
and a nats consumer. It also includes examples on how to
add health checks, prometheus metrics and migrations`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to execute root command")
	}
}

func init() { //nolint:gochecknoinits
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AddConfigPath(".")
	viper.AddConfigPath("/config")

	// First we load the default configs
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err == nil {
		logrus.Info("Using config file: ", viper.ConfigFileUsed())
	}

	// Then we load the secrets configs (if any) and merge it with the default configs
	viper.SetConfigName("secrets")
	err := viper.MergeInConfig()
	if err == nil {
		logrus.Info("Merging config file: ", viper.ConfigFileUsed())
	} else if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		logrus.Error("Missing config file 'secrets.yaml'. View readme for more information")
	}

	// Then we load the config file specified by the user (if any) --config flag and merge it with the default configs
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.MergeInConfig(); err == nil {
			logrus.Info("Merging config file: ", viper.ConfigFileUsed())
		}
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func loadConfig(configs ...string) (*Config, error) {
	var config Config
	defaults.SetDefaults(&config)
	bindEnvs(config)

	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	match := regexp.MustCompile(`.*`)
	if len(configs) != 0 {
		match = regexp.MustCompile(strings.ToLower(fmt.Sprintf("^Config.(%s)", strings.Join(configs, "|"))))
	}

	err = validator.New().StructFiltered(config, func(ns []byte) bool {
		return !match.MatchString(strings.ToLower(string(ns)))
	})
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func GetLogger(config LogConfig) *logrus.Logger {
	logger := logrus.New()
	lvl, err := logrus.ParseLevel(config.Level)
	if err != nil {
		lvl = logrus.InfoLevel
		logger.WithError(err).Warnf("Failed to parse log level, setting log level to '%s'", lvl)
	}
	logger.SetLevel(lvl)
	switch config.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	return logger
}

// Adapted from https://github.com/spf13/viper/issues/188#issuecomment-401431526
func bindEnvs(iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		fieldv := ifv.Field(i)
		t := ift.Field(i)
		name := strings.ToLower(t.Name)
		tag, ok := t.Tag.Lookup("mapstructure")
		if ok {
			name = tag
		}
		parts := append(parts, name)
		switch fieldv.Kind() { //nolint:exhaustive
		case reflect.Struct:
			bindEnvs(fieldv.Interface(), parts...)
		default:
			viper.BindEnv(strings.Join(parts, ".")) //nolint:errcheck
		}
	}
}
