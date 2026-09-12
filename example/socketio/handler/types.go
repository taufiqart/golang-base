package socketio

// Message represents an asynchronous real-time event sent via Go channel.
type Message struct {
	// Namespace is the target Socket.IO namespace (e.g. "/chat"). If empty, uses the root namespace "/".
	Namespace string `json:"namespace,omitempty"`

	// Room is the target room (e.g. "user:123", "room:general"). If empty, broadcasts to all clients in the namespace.
	Room string `json:"room,omitempty"`

	// Event is the Socket.IO event name (e.g. "notification:new", "order:created").
	Event string `json:"event"`

	// Payload is the data sent with the event. Must be serializable to JSON.
	Payload any `json:"payload"`
}
