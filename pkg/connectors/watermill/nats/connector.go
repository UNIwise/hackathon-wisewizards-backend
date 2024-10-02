package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/nats-io/stan.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uniwise/marshal"
	"github.com/uniwise/walrus"
)

type Config struct {
	DSN              string        `validate:"required" mapstructure:"dsn"`
	Cluster          string        `validate:"required" mapstructure:"cluster"`
	ConsumerID       string        `validate:"required" mapstructure:"consumer_id"`
	QueueGroup       string        `validate:"required" mapstructure:"queue_group"`
	DurableName      string        `validate:"required" mapstructure:"durable_name"`
	AckWaitTimeout   time.Duration `validate:"required" mapstructure:"ack_wait_timeout"`
	SubscribersCount int           `validate:"required" mapstructure:"subscribers_count"`
}

type Connector struct {
	config     Config
	logger     watermill.LoggerAdapter
	publisher  message.Publisher
	subscriber message.Subscriber
	client     *stan.Conn
	router     *message.Router
}

func NewConnector(config Config, log *logrus.Entry) (*Connector, error) {
	logger := walrus.NewWithLogger(log)

	client, err := createClient(config, log)
	if err != nil {
		log.WithError(err).Error("Failed to create NATS client")
		return nil, err
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create watermill router")
	}

	conn := &Connector{
		config: config,
		logger: logger,
		client: client,
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

func (conn *Connector) Subscribe(handlerName string, subscribeTopic string, handlerFunc message.NoPublishHandlerFunc) error {
	if conn.subscriber == nil {
		if err := conn.initSubscriber(); err != nil {
			return err
		}
	}

	conn.router.AddNoPublisherHandler(
		handlerName,
		subscribeTopic,
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

func (conn *Connector) initPublisher() error {
	publisher, err := nats.NewStreamingPublisherWithStanConn(
		*conn.client,
		nats.StreamingPublisherPublishConfig{
			Marshaler: marshal.NewMarshalerUnmarshaler(),
		},
		conn.logger,
	)
	if err != nil {
		return errors.Wrap(err, "Failed to create NATS publisher")
	}

	conn.publisher = publisher

	return nil
}

func (conn *Connector) initSubscriber() error {
	subscriber, err := nats.NewStreamingSubscriberWithStanConn(
		*conn.client,
		nats.StreamingSubscriberSubscriptionConfig{
			QueueGroup:       conn.config.QueueGroup,
			DurableName:      conn.config.DurableName,
			SubscribersCount: conn.config.SubscribersCount,
			CloseTimeout:     time.Minute,
			AckWaitTimeout:   conn.config.AckWaitTimeout,
			StanSubscriptionOptions: []stan.SubscriptionOption{
				stan.StartWithLastReceived(),
			},
			Unmarshaler: marshal.NewMarshalerUnmarshaler(),
		},
		conn.logger,
	)
	if err != nil {
		return errors.Wrap(err, "Failed to create NATS subscriber")
	}

	conn.subscriber = subscriber

	return nil
}

func createClient(config Config, log *logrus.Entry) (*stan.Conn, error) {
	sc, err := stan.Connect(
		config.Cluster,
		fmt.Sprintf("%s-%s", config.ConsumerID, uuid.New().String()),
		stan.NatsURL(config.DSN),
		stan.SetConnectionLostHandler(func(_ stan.Conn, err error) {
			log.WithError(err).Fatal("Connection lost")
		}))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to establish connection to NATS steaming server")
	}

	return &sc, nil
}
