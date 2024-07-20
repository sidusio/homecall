package notifications

import (
	"context"
	"firebase.google.com/go/v4/messaging"
)

type Service interface {
	SendNotification(ctx context.Context, notification *messaging.Message) error
}
