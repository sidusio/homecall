package directorynotifications

import (
	"context"
	"firebase.google.com/go/v4/messaging"
	"fmt"
	"os"
	"path"
	"time"
)

const (
	TopicsDirectory  = "topics"
	DevicesDirectory = "devices"
)

type Service struct {
	directory string
}

func NewService(directory string) (*Service, error) {
	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	err = os.Chmod(directory, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to change directory permissions: %w", err)
	}

	err = os.MkdirAll(path.Join(directory, TopicsDirectory), 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create topics directory: %w", err)
	}

	err = os.MkdirAll(path.Join(directory, DevicesDirectory), 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create devices directory: %w", err)
	}

	return &Service{
		directory: directory,
	}, nil
}

func (s *Service) SendNotification(ctx context.Context, notification *messaging.Message) error {
	if notification.Topic == "" && notification.Token == "" {
		return fmt.Errorf("notification must have a topic or token, condition notifications are not supported")
	}
	notificationJSON, err := notification.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	fileName := fmt.Sprintf("notificiation-%s.json", time.Now().Format(time.RFC3339Nano))

	if !exactlyOne(
		notification.Topic != "",
		notification.Token != "",
		notification.Condition != "",
	) {
		return fmt.Errorf("notification must have exactly one of topic, token, or condition")
	}
	switch {
	case notification.Topic != "":
		dir := path.Join(s.directory, TopicsDirectory, notification.Topic)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create topic directory: %w", err)
		}
		fileName = path.Join(dir, fileName)
	case notification.Token != "":
		dir := path.Join(s.directory, DevicesDirectory, notification.Token)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create device directory: %w", err)
		}
		fileName = path.Join(dir, fileName)
	case notification.Condition != "":
		return fmt.Errorf("condition notifications are not supported")
	}

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(notificationJSON)
	if err != nil {
		return fmt.Errorf("failed to write notification to file: %w", err)
	}

	return nil
}

func exactlyOne(b ...bool) bool {
	count := 0
	for _, v := range b {
		if v {
			count++
		}
	}
	return count == 1
}
