package main

import (
	"log"
	"os"
	"path/filepath"

	pkgsocketio "golang-base/internal/pkg/socketio"

	"github.com/gofiber/fiber/v3"
	"github.com/zishang520/socket.io/v2/socket"
)

func main() {
	ioServer, err := pkgsocketio.NewServer(&pkgsocketio.Config{})
	if err != nil {
		log.Fatalf("Failed to create Socket.IO server: %v", err)
	}

	chatNamespace := ioServer.Of("/chat")
	chatNamespace.On("connection", func(args ...any) {
		client := args[0].(*socket.Socket)
		log.Printf("[Socket.IO /chat] Client connected: %s", client.Id())

		client.On("chat:send", func(eventArgs ...any) {
			if len(eventArgs) > 0 {
				log.Printf("[Socket.IO /chat] Received message: %v", eventArgs[0])
				chatNamespace.Emit("chat:receive", eventArgs[0])
			}
		})

		client.On("disconnect", func(reason ...any) {
			log.Printf("[Socket.IO /chat] Client disconnected: %s", client.Id())
		})
	})

	app := fiber.New()

	ioServer.Register(app)

	dir, _ := os.Getwd()
	htmlPath := filepath.Join(dir, "example", "socketio", "chat", "index.html")
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(htmlPath)
	})

	log.Println("==================================================")
	log.Println("🚀 Chat demo UI  : http://localhost:3150")
	log.Println("   Socket.IO     : ws://localhost:3150/socket.io/")
	log.Println("   Open multiple tabs to test!")
	log.Println("==================================================")

	if err := app.Listen(":3150"); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}
