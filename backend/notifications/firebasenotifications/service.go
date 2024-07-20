package firebasenotifications

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"fmt"
)

type Service struct {
	messageClient *messaging.Client
}

func NewService(ctx context.Context, firebaseProjectId string) (*Service, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: firebaseProjectId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase app: %w", err)
	}

	messageClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create messaging client: %w", err)
	}

	return &Service{
		messageClient: messageClient,
	}, nil
}

func (s *Service) SendNotification(ctx context.Context, notification *messaging.Message) error {
	_, err := s.messageClient.Send(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}
