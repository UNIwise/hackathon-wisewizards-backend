//go:generate mockgen --source=connector.go -destination=connector_mock.go -package=watermill -mock_names Service=MockConnector
package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type Connector interface {
	Subscribe(handlerName string, subscribeTopic string, handlerFunc message.NoPublishHandlerFunc) error
	Publish(topic string, messages ...*message.Message) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
