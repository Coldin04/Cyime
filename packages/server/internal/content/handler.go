package content

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetContentHandler handles GET /api/v1/workspace/markdowns/:id/content
func GetContentHandler(c *fiber.Ctx) error {
	// Get user ID from locals (set by Protected middleware)
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid User ID",
			Message: "User ID format is invalid",
		})
	}

	// Parse markdown ID from path
	markdownIDStr := c.Params("id")
	markdownID, err := uuid.Parse(markdownIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Markdown ID",
			Message: "Markdown ID must be a valid UUID",
		})
	}

	// Get content
	result, err := GetContent(userID, markdownID)
	if err != nil {
		switch err.Error() {
		case "文档不存在或无权访问", "文档内容不存在":
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "Not Found",
				Message: err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}
	}

	return c.JSON(result)
}

// GetContentByVersionHandler handles GET /api/v1/workspace/markdowns/:id/versions/:version
func GetContentByVersionHandler(c *fiber.Ctx) error {
	// Get user ID from locals
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid User ID",
			Message: "User ID format is invalid",
		})
	}

	// Parse markdown ID from path
	markdownIDStr := c.Params("id")
	markdownID, err := uuid.Parse(markdownIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Markdown ID",
			Message: "Markdown ID must be a valid UUID",
		})
	}

	// Parse version from path
	version, err := c.ParamsInt("version")
	if err != nil || version <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Version must be a positive integer",
		})
	}

	// Get content by version
	result, err := GetContentByVersion(userID, markdownID, version)
	if err != nil {
		switch err.Error() {
		case "文档不存在或无权访问", "指定版本的内容不存在":
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "Not Found",
				Message: err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}
	}

	return c.JSON(result)
}

// GetVersionsHandler handles GET /api/v1/workspace/markdowns/:id/versions
func GetVersionsHandler(c *fiber.Ctx) error {
	// Get user ID from locals
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid User ID",
			Message: "User ID format is invalid",
		})
	}

	// Parse markdown ID from path
	markdownIDStr := c.Params("id")
	markdownID, err := uuid.Parse(markdownIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Markdown ID",
			Message: "Markdown ID must be a valid UUID",
		})
	}

	// Get versions
	versions, err := GetVersions(userID, markdownID)
	if err != nil {
		switch err.Error() {
		case "文档不存在或无权访问":
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "Not Found",
				Message: err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"versions": versions,
	})
}

// UpdateContentRequest represents the request body for updating content
type UpdateContentRequest struct {
	Content string `json:"content"`
}

// UpdateContentHandler handles PUT /api/v1/workspace/markdowns/:id/content
func UpdateContentHandler(c *fiber.Ctx) error {
	// Get user ID from locals
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid User ID",
			Message: "User ID format is invalid",
		})
	}

	// Parse markdown ID from path
	markdownIDStr := c.Params("id")
	markdownID, err := uuid.Parse(markdownIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Markdown ID",
			Message: "Markdown ID must be a valid UUID",
		})
	}

	// Parse request body
	var req UpdateContentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid request body",
		})
	}

	// Validate content
	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "内容不能为空",
		})
	}

	// Update content
	result, err := UpdateContent(userID, markdownID, req.Content)
	if err != nil {
		switch err.Error() {
		case "文档不存在或无权访问":
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "Not Found",
				Message: err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"success":   result.Success,
		"version":   result.Version,
		"updatedAt": result.UpdatedAt,
	})
}
