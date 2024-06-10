package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

const (
	callsTopic       = "homecall.calls"
	enrollmentsTopic = "homecall.enrollments"
)

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

	enrollmentBroadcaster, err := gochannel.NewFanOut(baseChannel, wLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create enrollment broadcaster: %w", err)
	}
	enrollmentBroadcaster.AddSubscription(enrollmentsTopic)

	return &Broker{
		baseChannel:           baseChannel,
		callBroadcaster:       callBroadcaster,
		enrollmentBroadcaster: enrollmentBroadcaster,
		started:               make(chan struct{}),
	}, nil
}

type Broker struct {
	baseChannel           pubSub
	callBroadcaster       *gochannel.FanOut
	enrollmentBroadcaster *gochannel.FanOut
	started               chan struct{}
}

func (b *Broker) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		err := b.callBroadcaster.Run(ctx)
		if err != nil {
			return fmt.Errorf("failed to run call broadcaster: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		err := b.enrollmentBroadcaster.Run(ctx)
		if err != nil {
			return fmt.Errorf("failed to run enrollment broadcaster: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		<-b.callBroadcaster.Running()
		<-b.enrollmentBroadcaster.Running()
		close(b.started)
		return nil
	})

	err := eg.Wait()
	if err != nil {
		return fmt.Errorf("broker exited: %w", err)
	}
	return nil
}

func (b *Broker) Close() error {

	err := b.enrollmentBroadcaster.Close()
	if err != nil {
		return fmt.Errorf("failed to close enrollment-broadcaster: %w", err)
	}

	err = b.callBroadcaster.Close()
	if err != nil {
		return fmt.Errorf("failed to close call-broadcaster: %w", err)
	}

	err = b.baseChannel.Close()
	if err != nil {
		return fmt.Errorf("failed to close baseChannel: %w", err)
	}

	return nil
}

func (b *Broker) Started() <-chan struct{} {
	return b.started
}

func (b *Broker) PublishCall(call Call) error {
	callBytes, err := json.Marshal(call)
	if err != nil {
		return fmt.Errorf("failed to marshal call: %w", err)
	}

	return b.baseChannel.Publish(callsTopic, message.NewMessage(watermill.NewULID(), callBytes))
}

func (b *Broker) SubscribeToCalls(ctx context.Context, deviceId string, handler func(Call) error) error {
	messages, err := b.callBroadcaster.Subscribe(ctx, callsTopic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to calls: %w", err)
	}

	for msg := range messages {
		var call Call
		err := json.Unmarshal(msg.Payload, &call)
		if err != nil {
			msg.Nack()
			return fmt.Errorf("failed to unmarshal call: %w", err)
		}

		if call.DeviceID != deviceId {
			msg.Ack()
			continue
		}

		err = handler(call)
		if err != nil {
			msg.Nack()
			return fmt.Errorf("failed to handle call: %w", err)
		}
		msg.Ack()
	}
	return nil
}

func (b *Broker) PublishEnrollment(deviceId string) error {
	return b.baseChannel.Publish(enrollmentsTopic, message.NewMessage(watermill.NewULID(), []byte(deviceId)))
}

func (b *Broker) SubscribeToEnrollment(ctx context.Context, deviceId string, handler func() error) error {
	messages, err := b.enrollmentBroadcaster.Subscribe(ctx, enrollmentsTopic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to enrollments: %w", err)
	}

	for msg := range messages {

		if string(msg.Payload) != deviceId {
			msg.Ack()
			continue
		}

		err = handler()
		if err != nil {
			msg.Nack()
			return fmt.Errorf("failed to handle enrollment: %w", err)
		}
		msg.Ack()
	}
	return nil
}
