package main

import (
	"context"
	"log"
	"os"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/auth"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/config"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/content"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/media"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/middleware"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/user"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/workspace"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	_ = config.LoadDotEnv(".env")

	// Initialize database
	database.Connect()
	log.Println("Database initialization complete.")
	media.StartAssetGCWorker(context.Background())

	// Create new Fiber app
	app := fiber.New()
	app.Use(recover.New())

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
	workspaceRoutes.Get("/files/:id", workspace.GetFileHandler)
	workspaceRoutes.Post("/folders", workspace.CreateFolderHandler)
	workspaceRoutes.Post("/documents", workspace.CreateDocumentHandler)
	workspaceRoutes.Post("/files/batch-delete", workspace.BatchDeleteHandler)
	workspaceRoutes.Delete("/files/:id", workspace.DeleteFileHandler)
	workspaceRoutes.Get("/folders/:id/ancestors", workspace.GetFolderAncestorsHandler)
	workspaceRoutes.Get("/trash", workspace.GetTrashHandler)
	workspaceRoutes.Post("/trash/restore", workspace.RestoreTrashHandler)
	workspaceRoutes.Delete("/trash", workspace.PermanentDeleteHandler)
	// Update document title
	workspaceRoutes.Put("/documents/:id/title", workspace.UpdateDocumentTitleHandler)
	// Update folder name
	workspaceRoutes.Put("/folders/:id/name", workspace.UpdateFolderNameHandler)
	// Move document
	workspaceRoutes.Put("/documents/:id/move", workspace.MoveDocumentHandler)
	// Move folder
	workspaceRoutes.Put("/folders/:id/move", workspace.MoveFolderHandler)
	// Batch move files and folders
	workspaceRoutes.Post("/files/batch-move", workspace.BatchMoveHandler)

	// Edit routes (protected) - for document content management
	editRoutes := api.Group("/edit/documents", middleware.Protected())
	editRoutes.Get("/:id/content", content.GetContentHandler)
	editRoutes.Put("/:id/content", content.UpdateContentHandler)
	editRoutes.Post("/:id/assets", media.UploadDocumentAssetHandler)

	// Media read routes:
	// - URL exchange is protected by JWT.
	// - Content endpoint is public but protected by short-lived media token.
	api.Get("/media/assets", middleware.Protected(), media.ListAssetsHandler)
	api.Get("/media/assets/:id/url", middleware.Protected(), media.GetAssetURLHandler)
	api.Get("/media/assets/:id/references", middleware.Protected(), media.GetAssetReferencesHandler)
	api.Delete("/media/assets/:id", middleware.Protected(), media.DeleteAssetHandler)
	api.Get("/media/assets/:id/content", media.GetAssetContentHandler)

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
