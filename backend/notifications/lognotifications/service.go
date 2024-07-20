package lognotifications

import (
	"context"
	"firebase.google.com/go/v4/messaging"
	"log/slog"
)

type Service struct {
	log *slog.Logger
}

func NewService(log *slog.Logger) *Service {
	return &Service{
		log: log,
	}
}

func (s *Service) SendNotification(ctx context.Context, notification *messaging.Message) error {
	s.log.Info("sending notification", slog.Any("notification", notification))
	return nil
}
