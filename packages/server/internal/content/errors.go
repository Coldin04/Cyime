package content

import "errors"

var (
	ErrDocumentNotFoundOrUnauthorized = errors.New("文档不存在或无权访问")
	ErrDocumentContentNotFound        = errors.New("文档内容不存在")
	ErrInvalidContentJSON             = errors.New("contentJson must be valid JSON")
	ErrInvalidContentAssetReferences  = errors.New("content references invalid assets")
)
