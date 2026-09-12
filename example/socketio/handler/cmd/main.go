package main

import (
	"log"
	"os"

	socketiomod "golang-base/example/socketio/handler"

	"github.com/gofiber/fiber/v3"
	"github.com/zishang520/socket.io/v2/socket"
)

func main() {
	mod, err := socketiomod.New(nil, nil)
	if err != nil {
		log.Fatalf("Failed to initialize Socket.IO module: %v", err)
	}
	defer mod.Close()

	app := fiber.New()

	mod.RegisterSocket(app)
	mod.Register(app.Group("/api/v1"))

	registerChatEvents(mod)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3160"
	}

	log.Printf("Socket.IO handler example running on http://localhost:%s", port)
	log.Printf("  POST /api/v1/socketio/broadcast  (REST broadcast endpoint)")
	log.Printf("  WS   ws://localhost:%s/socket.io/  (event listener)", port)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

func registerChatEvents(mod *socketiomod.Module) {
	chat := mod.Server().Of("/chat")
	chat.On("connection", func(args ...any) {
		client := args[0].(*socket.Socket)
		client.Join(socket.Room("room:general"))
		log.Printf("[event] client connected: %s", client.Id())

		client.On("chat:send", func(eventArgs ...any) {
			if len(eventArgs) == 0 {
				return
			}
			log.Printf("[event] chat:send from %s", client.Id())
			chat.To(socket.Room("room:general")).Emit("chat:receive", eventArgs[0])
		})

		client.On("disconnect", func(reason ...any) {
			log.Printf("[event] client disconnected: %s", client.Id())
		})
	})
}
