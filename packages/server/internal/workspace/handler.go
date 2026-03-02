package workspace

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetFilesHandler handles GET /api/v1/workspace/files
func GetFilesHandler(c *fiber.Ctx) error {
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

	// Parse query parameters
	parentIDStr := c.Query("parent_id")
	var parentID *uuid.UUID
	if parentIDStr != "" && parentIDStr != "null" {
		pid, err := uuid.Parse(parentIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "Invalid parent_id",
				Message: "parent_id must be a valid UUID or 'null'",
			})
		}
		parentID = &pid
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	sortBy := c.Query("sort_by", "updated_at")
	order := c.Query("order", "desc")
	filterType := c.Query("type", "all")

	// Get files
	response, err := GetFiles(userID, parentID, limit, offset, sortBy, order, filterType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(response)
}

// CreateFolderHandler handles POST /api/v1/workspace/folders
func CreateFolderHandler(c *fiber.Ctx) error {
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

	// Parse request body
	var req CreateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid request body",
		})
	}

	// Create folder
	folder, err := CreateFolder(userID, req.Name, req.Description, req.ParentID)
	if err != nil {
		switch err.Error() {
		case "文件夹名称不能为空", "文件夹名称不能超过 255 个字符", "不能使用系统保留的文件夹名称":
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "Validation Error",
				Message: err.Error(),
			})
		case "父文件夹不存在", "同名文件夹已存在":
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

	// Get creator info
	creatorInfo, err := GetCreatorInfo(userID)
	if err != nil {
		creatorInfo = &CreatorInfo{
			ID:          userID,
			DisplayName: nil,
		}
	}

	// Build response
	response := CreateFolderResponse{
		ID:          folder.ID,
		Type:        "folder",
		Name:        folder.Name,
		Description: folder.Description,
		ParentID:    folder.ParentID,
		CreatedAt:   folder.CreatedAt,
		UpdatedAt:   folder.UpdatedAt,
		Creator:     *creatorInfo,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// CreateMarkdownHandler handles POST /api/v1/workspace/markdowns
func CreateMarkdownHandler(c *fiber.Ctx) error {
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

	// Parse request body
	var req CreateMarkdownRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid request body",
		})
	}

	// Create markdown
	markdown, err := CreateMarkdown(userID, req.Title, req.Content, req.FolderID)
	if err != nil {
		switch err.Error() {
		case "文档标题不能为空", "文档标题不能超过 255 个字符":
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "Validation Error",
				Message: err.Error(),
			})
		case "文件夹不存在":
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

	// Get creator info
	creatorInfo, err := GetCreatorInfo(userID)
	if err != nil {
		creatorInfo = &CreatorInfo{
			ID:          userID,
			DisplayName: nil,
		}
	}

	// Build response
	response := CreateMarkdownResponse{
		ID:        markdown.ID,
		Type:      "markdown",
		Title:     markdown.Title,
		Excerpt:   markdown.Excerpt,
		FolderID:  markdown.FolderID,
		CreatedAt: markdown.CreatedAt,
		UpdatedAt: markdown.UpdatedAt,
		Creator:   *creatorInfo,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// DeleteFileHandler handles DELETE /api/v1/workspace/files/:id
func DeleteFileHandler(c *fiber.Ctx) error {
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

	// Parse file ID
	fileIDStr := c.Params("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid File ID",
			Message: "File ID must be a valid UUID",
		})
	}

	// Get file type from query parameter
	fileType := c.Query("type")
	if fileType != "folder" && fileType != "markdown" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "type parameter must be either 'folder' or 'markdown'",
		})
	}

	// Delete file
	if err := DeleteFile(userID, fileID, fileType); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	return c.JSON(DeleteResponse{
		Success: true,
		Message: "文件已移动到回收站",
	})
}

// GetFolderAncestorsHandler handles GET /api/v1/workspace/folders/:id/ancestors
func GetFolderAncestorsHandler(c *fiber.Ctx) error {
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

	// Parse folder ID from path
	folderIDStr := c.Params("id")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Folder ID",
			Message: "Folder ID must be a valid UUID",
		})
	}

	// Get ancestors from service
	ancestors, err := GetFolderAncestors(userID, folderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(ancestors)
}

