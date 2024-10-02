package rabbitmq

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uniwise/walrus"
)

type PubSubConfig struct {
	QueueName  string
	RoutingKey string
}

type Config struct {
	URL string `validate:"required" mapstructure:"url"`

	// Router middleware
	MaxRetries      int           `validate:"required" mapstructure:"max_retries"`
	InitialInterval time.Duration `validate:"required" mapstructure:"initial_interval"`
	Multiplier      float64       `validate:"required" mapstructure:"multiplier"`
	MaxElapsedTime  time.Duration `validate:"required" mapstructure:"max_elapsed_time"`
	MaxInterval     time.Duration `validate:"required" mapstructure:"max_interval"`

	SubscriberConfig PubSubConfig
	PublisherConfig  PubSubConfig
}

type Connector struct {
	config     Config
	logger     watermill.LoggerAdapter
	subscriber message.Subscriber
	publisher  message.Publisher
	router     *message.Router
}

func NewConnector(config Config, log *logrus.Entry) (*Connector, error) {
	logger := walrus.NewWithLogger(log)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create watermill router")
	}

	router.AddPlugin(plugin.SignalsHandler)

	router.AddMiddleware(middleware.Retry{
		MaxRetries:      config.MaxRetries,
		InitialInterval: config.InitialInterval,
		Multiplier:      config.Multiplier,
		MaxElapsedTime:  config.MaxElapsedTime,
		MaxInterval:     config.MaxInterval,
		Logger:          logger,
	}.Middleware)

	conn := &Connector{
		config: config,
		logger: logger,
		router: router,
	}

	return conn, nil
}

func (conn *Connector) Publish(topic string, messages ...*message.Message) error {
	if conn.publisher == nil {
		if err := conn.initPublisher(); err != nil {
			return err
		}
	}

	if err := conn.publisher.Publish(topic, messages...); err != nil {
		return errors.Wrap(err, "Failed to publish message")
	}

	return nil
}

func (conn *Connector) Subscribe(handlerName string, topic string, handlerFunc message.NoPublishHandlerFunc) error {
	if conn.subscriber == nil {
		if err := conn.initSubscriber(); err != nil {
			return err
		}
	}

	conn.router.AddNoPublisherHandler(
		handlerName,
		topic,
		conn.subscriber,
		handlerFunc,
	)

	return nil
}

func (conn *Connector) Start(ctx context.Context) error {
	if err := conn.router.Run(ctx); err != nil {
		return errors.Wrap(err, "Failed to start router")
	}

	return nil
}

func (conn *Connector) Stop(ctx context.Context) error {
	if err := conn.router.Close(); err != nil {
		return errors.Wrap(err, "Failed to close router")
	}

	return nil
}

func (conn *Connector) initSubscriber() error {
	if conn.config.SubscriberConfig == (PubSubConfig{}) {
		return errors.New("Subscriber config is not set")
	}

	amqpConfig := createAMQPConfig(
		conn.config.URL,
		conn.config.SubscriberConfig.QueueName,
		conn.config.SubscriberConfig.RoutingKey,
	)

	subscriber, err := amqp.NewSubscriber(amqpConfig, conn.logger)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize RabbitMQ subscriber")
	}

	conn.subscriber = subscriber

	return nil
}

func (conn *Connector) initPublisher() error {
	if conn.config.PublisherConfig == (PubSubConfig{}) {
		return errors.New("Publisher config is not set")
	}

	amqpConfig := createAMQPConfig(
		conn.config.URL,
		conn.config.PublisherConfig.QueueName,
		conn.config.PublisherConfig.RoutingKey,
	)

	publisher, err := amqp.NewPublisher(amqpConfig, conn.logger)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize RabbitMQ publisher")
	}

	ch, err := publisher.Connection().Channel()
	if err != nil {
		return errors.Wrap(err, "Failed to open channel for RabbitMQ publisher")
	}

	if err = amqpConfig.TopologyBuilder.BuildTopology(
		ch,
		conn.config.PublisherConfig.QueueName,
		conn.config.PublisherConfig.QueueName,
		amqpConfig, conn.logger,
	); err != nil {
		return errors.Wrap(err, "Failed to build topology for RabbitMQ publisher")
	}

	conn.publisher = publisher

	return nil
}

func createAMQPConfig(amqpURI, queueName, routingKey string) amqp.Config {
	return amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: amqpURI,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: func(topic string) string {
				return topic
			},
			Type:    "direct",
			Durable: true,
		},
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameConstant(queueName),
			Durable:      true,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return routingKey
			},
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return routingKey
			},
		},
		Consume: amqp.ConsumeConfig{
			Qos: amqp.QosConfig{
				PrefetchCount: 1,
			},
			NoRequeueOnNack: true,
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
}
