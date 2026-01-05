package main

import (
	"log"
	"os"

	"faucet-backend/config"
	"faucet-backend/database"
	"faucet-backend/handlers"
	"faucet-backend/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env in development
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		godotenv.Load()
	}

	// Initialize database
	database.Connect()
	database.Migrate()

	// Initialize Redis
	database.ConnectRedis()

	// Seed default tokens
	config.SeedTokens()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(middleware.CORS())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"wallet": handlers.GetFaucetAddress(),
		})
	})

	// API routes
	api := app.Group("/api")

	faucet := api.Group("/faucet")
	faucet.Post("/drip", handlers.RequestDrip)
	faucet.Get("/status/:address", handlers.GetStatus)
	faucet.Get("/tokens", handlers.GetTokens)
	faucet.Get("/stats", handlers.GetStats)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Multi-token faucet backend starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
