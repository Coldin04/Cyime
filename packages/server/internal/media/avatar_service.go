package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type UploadUserAvatarResult struct {
	URL       string
	ObjectKey string
}

func UploadUserAvatar(ctx context.Context, userID uuid.UUID, fileHeader *multipart.FileHeader) (*UploadUserAvatarResult, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("file is required")
	}
	contentType, ok := normalizeAllowedContentType(
		strings.TrimSpace(fileHeader.Header.Get("Content-Type")),
		fileHeader.Filename,
	)
	if !ok || !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("unsupported avatar file type: %s", contentType)
	}
	if err := initStorageProvider(); err != nil {
		return nil, err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	objectKey := buildUserAvatarObjectKey(userID, fileHeader.Filename)
	uploadResult, err := storageProvider.PutObject(ctx, PutObjectInput{
		ObjectKey:   objectKey,
		ContentType: contentType,
		Body:        bytes.NewReader(fileBytes),
	})
	if err != nil {
		return nil, err
	}

	return &UploadUserAvatarResult{
		URL:       uploadResult.URL,
		ObjectKey: objectKey,
	}, nil
}

func DeleteStoredObject(ctx context.Context, objectKey string) error {
	if strings.TrimSpace(objectKey) == "" {
		return nil
	}
	if err := initStorageProvider(); err != nil {
		return err
	}
	return storageProvider.DeleteObject(ctx, objectKey)
}

func buildUserAvatarObjectKey(userID uuid.UUID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	return fmt.Sprintf("%s/avatars/%s%s", userID.String(), uuid.NewString(), ext)
}

func ResetStorageProviderForTesting() {
	storageProvider = nil
}

func InitStorageProviderForAvatarRead() error {
	return initStorageProvider()
}

func GetStoredObject(ctx context.Context, objectKey string) (*GetObjectResult, error) {
	if err := initStorageProvider(); err != nil {
		return nil, err
	}
	return storageProvider.GetObject(ctx, objectKey)
}
