package websocket

import "github.com/google/uuid"

type NotificationMessage struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Level   string `json:"level"`
}

func NewNotification(Type, Title, Content string) *NotificationMessage {

	return &NotificationMessage{
		Type:    Type,
		ID:      uuid.New().String(),
		Title:   Title,
		Content: Content,
		Level:   "success",
	}
}
