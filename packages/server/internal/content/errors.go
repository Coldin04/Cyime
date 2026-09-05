package content

import (
	"errors"
	"fmt"
)

var (
	ErrDocumentNotFoundOrUnauthorized = errors.New("文档不存在或无权访问")
	ErrDocumentContentNotFound        = errors.New("文档内容不存在")
	ErrInvalidContentJSON             = errors.New("contentJson must be valid JSON")
	ErrContentJSONTooLarge            = errors.New("contentJson exceeds maximum size")
	ErrInvalidContentAssetReferences  = errors.New("content references invalid assets")
	ErrWorkspaceStorageQuotaExceeded  = errors.New("已达到工作区存储空间上限")
	ErrInvalidContentVersion          = errors.New("expectedContentVersion must be a positive integer")
)

type ContentVersionConflictError struct {
	CurrentVersion int64
}

func (e *ContentVersionConflictError) Error() string {
	return fmt.Sprintf("document content changed since it was loaded (current version %d)", e.CurrentVersion)
}
