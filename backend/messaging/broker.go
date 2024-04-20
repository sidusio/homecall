package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"log/slog"
)

const callsTopic = "homecall.calls"

type pubSub interface {
	message.Publisher
	message.Subscriber
}

func NewBroker(logger *slog.Logger) (*Broker, error) {
	wLogger := watermill.NewSlogLogger(logger.With("library", "watermill"))

	baseChannel := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer:            0,
		Persistent:                     false,
		BlockPublishUntilSubscriberAck: false,
	}, wLogger)

	callBroadcaster, err := gochannel.NewFanOut(baseChannel, wLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create call broadcaster: %w", err)
	}
	callBroadcaster.AddSubscription(callsTopic)

	return &Broker{
		baseChannel:     baseChannel,
		callBroadcaster: callBroadcaster,
	}, nil
}

type Broker struct {
	baseChannel     pubSub
	callBroadcaster *gochannel.FanOut
}

func (b *Broker) Run(ctx context.Context) error {
	err := b.callBroadcaster.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run call-broadcaster: %w", err)
	}
	return nil
}

func (b *Broker) Close() error {
	err := b.baseChannel.Close()
	if err != nil {
		return fmt.Errorf("failed to close baseChannel: %w", err)
	}

	err = b.callBroadcaster.Close()
	if err != nil {
		return fmt.Errorf("failed to close call-broadcaster: %W", err)
	}
	return nil
}

func (b *Broker) PublishCall(call Call) error {
	callBytes, err := json.Marshal(call)
	if err != nil {
		return fmt.Errorf("failed to marshal call: %w", err)
	}

	return b.baseChannel.Publish(callsTopic, message.NewMessage(watermill.NewULID(), callBytes))
}

func (b *Broker) SubscribeToCalls(ctx context.Context, handler func(Call) error) error {
	messages, err := b.callBroadcaster.Subscribe(ctx, callsTopic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to calls: %w", err)
	}

	for msg := range messages {
		var call Call
		err := json.Unmarshal(msg.Payload, &call)
		if err != nil {
			return fmt.Errorf("failed to unmarshal call: %w", err)
		}

		err = handler(call)
		if err != nil {
			return fmt.Errorf("failed to handle call: %w", err)
		}
	}
	return nil
}
