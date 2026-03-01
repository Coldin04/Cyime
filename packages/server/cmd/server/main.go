package main

import (
	"log"
	"os"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/auth"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/middleware"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/user"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/workspace"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	// Initialize database
	database.Connect()
	log.Println("Database initialization complete.")

	// Create new Fiber app
	app := fiber.New()

	// Add flexible CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
			if allowedOrigins == "" {
				// Default for local development
				return origin == "http://localhost:5173"
			}
			for _, allowed := range strings.Split(allowedOrigins, ",") {
				if origin == allowed {
					return true
				}
			}
			return false
		},
		AllowCredentials: true,
	}))

	// --- ROUTING ---
	api := app.Group("/api/v1")

	// Auth routes
	authRoutes := api.Group("/auth")
		authRoutes.Get("/config", auth.GetAuthConfig)
		authRoutes.Get("/login/:provider", auth.AuthLogin)
		authRoutes.Get("/callback/:provider", auth.AuthCallback)
		authRoutes.Post("/refresh", auth.HandleRefresh)
		authRoutes.Post("/logout", auth.HandleLogout)
	
		// User routes (protected)
		userRoutes := api.Group("/user", middleware.Protected())
		userRoutes.Get("/me", user.GetMe)

		// Workspace routes (protected)
		workspaceRoutes := api.Group("/workspace", middleware.Protected())
		workspaceRoutes.Get("/files", workspace.GetFilesHandler)
		workspaceRoutes.Post("/folders", workspace.CreateFolderHandler)
		workspaceRoutes.Post("/markdowns", workspace.CreateMarkdownHandler)
		workspaceRoutes.Delete("/files/:id", workspace.DeleteFileHandler)
	


	// Simple root route to check if server is up
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello from CyimeWrite Server!")
	})

	log.Println("Starting server on port 8080...")
	// Start server
	if err := app.Listen(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

