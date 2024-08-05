package jitsi

import "fmt"

type Call struct {
	roomName string
	app      *App
}

func (c *Call) OfficeJWT(displayName string) (string, error) {
	token, err := c.app.jitsiJWT(c.roomName, displayName, "office")
	if err != nil {
		return "", fmt.Errorf("failed to create office JWT: %w", err)
	}
	return token, nil
}

func (c *Call) DeviceJWT(displayName string) (string, error) {
	token, err := c.app.jitsiJWT(c.roomName, displayName, "device")
	if err != nil {
		return "", fmt.Errorf("failed to create device JWT: %w", err)
	}
	return token, nil
}

func (c *Call) RoomName() string {
	return fmt.Sprintf("%s/%s", c.app.appId, c.roomName)
}
