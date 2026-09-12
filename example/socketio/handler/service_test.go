package socketio

import (
	"testing"
	"time"

	pkgsocketio "golang-base/internal/pkg/socketio"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_PublishAndWorker(t *testing.T) {
	server, err := pkgsocketio.NewServer(nil)
	require.NoError(t, err)
	defer server.Close()

	svc := NewService(server, 10)
	svc.Start()
	defer svc.Stop()

	// Test direct channel push
	ch := svc.Channel()
	assert.NotNil(t, ch)
	ch <- Message{
		Room:    "room-channel",
		Event:   "test:event",
		Payload: map[string]string{"foo": "bar"},
	}

	// Test Publish helper
	ok := svc.Publish(Message{
		Room:    "room-publish",
		Event:   "test:publish",
		Payload: "hello",
	})
	assert.True(t, ok)

	// Test SendToUser helper
	ok = svc.SendToUser("user-123", "notification:new", map[string]string{"msg": "hi"})
	assert.True(t, ok)

	// Test Broadcast helper
	ok = svc.Broadcast("system:alert", "maintenance soon")
	assert.True(t, ok)

	// Test BroadcastToRoom helper
	ok = svc.BroadcastToRoom("orders", "order:created", 999)
	assert.True(t, ok)

	// Allow worker time to drain messages
	time.Sleep(50 * time.Millisecond)
}

func TestService_DispatchNamespace(t *testing.T) {
	server, err := pkgsocketio.NewServer(nil)
	require.NoError(t, err)
	defer server.Close()

	svc := NewService(server, 10)
	svc.Start()
	defer svc.Stop()

	impl := svc.(*service)

	cases := []struct {
		name string
		msg  Message
	}{
		{
			name: "root namespace room",
			msg:  Message{Room: "room-root", Event: "evt", Payload: "a"},
		},
		{
			name: "custom namespace room",
			msg:  Message{Namespace: "/chat", Room: "room:general", Event: "evt", Payload: "b"},
		},
		{
			name: "custom namespace broadcast",
			msg:  Message{Namespace: "/chat", Event: "evt", Payload: "c"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() { impl.dispatch(tc.msg) })
		})
	}
}

func TestModule_Lifecycle(t *testing.T) {
	mod, err := New(nil, nil)
	require.NoError(t, err)
	require.NotNil(t, mod)
	defer mod.Close()

	assert.NotNil(t, mod.Service())
	assert.NotNil(t, mod.Server())

	// Test package-level channel and helper functions
	assert.NotNil(t, Channel())

	ok := Publish(Message{
		Event:   "ping",
		Payload: "pong",
	})
	assert.True(t, ok)

	ok = SendToUser("user-abc", "welcome", "welcome to platform")
	assert.True(t, ok)

	ok = Broadcast("general", "welcome all")
	assert.True(t, ok)

	ok = BroadcastToRoom("chat", "message", "test")
	assert.True(t, ok)
}
