package socketio

// BroadcastRequest represents the payload for triggering a real-time event via HTTP.
type BroadcastRequest struct {
	Namespace string `json:"namespace,omitempty"`
	Event     string `json:"event" validate:"required"`
	Room      string `json:"room,omitempty"`
	Payload   any    `json:"payload" validate:"required"`
}
