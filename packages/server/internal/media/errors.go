package media

import (
	"errors"
	"fmt"
)

var (
	ErrAssetNotFoundOrForbidden = errors.New("资源不存在或无权访问")
	ErrInvalidAssetStatus       = errors.New("invalid asset status")
	ErrInvalidAssetKind         = errors.New("invalid asset kind")
	ErrAssetStillReferenced     = errors.New("asset is still referenced by documents")
	ErrAssetAlreadyDeleted      = errors.New("asset already deleted")
	ErrFileRequired             = errors.New("file is required")
	ErrInvalidVisibility        = errors.New("invalid visibility")
	ErrDocumentNotAccessible    = errors.New("文档不存在或无权访问")
)

type UnsupportedFileTypeError struct {
	ContentType string
}

func (e *UnsupportedFileTypeError) Error() string {
	return fmt.Sprintf("unsupported file type: %s", e.ContentType)
}

type UnsupportedAvatarFileTypeError struct {
	ContentType string
}

func (e *UnsupportedAvatarFileTypeError) Error() string {
	return fmt.Sprintf("unsupported avatar file type: %s", e.ContentType)
}
