package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const defaultAvatarMaxBytes int64 = 2 * 1024 * 1024

var ErrAvatarFileTooLarge = errors.New("avatar file too large")

type UploadUserAvatarResult struct {
	URL       string
	ObjectKey string
}

func UploadUserAvatar(ctx context.Context, userID uuid.UUID, fileHeader *multipart.FileHeader) (*UploadUserAvatarResult, error) {
	if fileHeader == nil {
		return nil, ErrFileRequired
	}
	contentType, ok := normalizeAllowedContentType(
		strings.TrimSpace(fileHeader.Header.Get("Content-Type")),
		fileHeader.Filename,
	)
	if !ok || !strings.HasPrefix(contentType, "image/") {
		return nil, &UnsupportedAvatarFileTypeError{ContentType: contentType}
	}
	maxBytes := avatarMaxBytes()
	if fileHeader.Size > 0 && fileHeader.Size > maxBytes {
		return nil, fmt.Errorf("%w: max %d bytes", ErrAvatarFileTooLarge, maxBytes)
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
	if int64(len(fileBytes)) > maxBytes {
		return nil, fmt.Errorf("%w: max %d bytes", ErrAvatarFileTooLarge, maxBytes)
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

func avatarMaxBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("MEDIA_AVATAR_MAX_BYTES"))
	if raw == "" {
		return defaultAvatarMaxBytes
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return defaultAvatarMaxBytes
	}
	return parsed
}
