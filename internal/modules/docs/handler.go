package docs

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

type handler struct{}

func newHandler() *handler {
	return &handler{}
}

func (h *handler) RegisterRoutes(router fiber.Router) {
	docsGroup := router.Group("/docs")
	docsGroup.Get("", h.SwaggerUI)
	docsGroup.Get("/openapi.yml", h.OpenAPISpec)
}

func (h *handler) SwaggerUI(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(swaggerUIHTML)
}

func (h *handler) OpenAPISpec(c *fiber.Ctx) error {
	data, err := os.ReadFile("docs/blueprint/openapi.yml")
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "openapi.yml not found",
		})
	}
	c.Set("Content-Type", "application/yaml")
	return c.Send(data)
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>Golang API Starter Kit - API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/api/v1/docs/openapi.yml",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
      tryItOutEnabled: true,
      persistAuthorization: true
    });
  </script>
</body>
</html>`
