package media

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PutObjectInput struct {
	ObjectKey   string
	ContentType string
	Body        io.Reader
}

type PutObjectResult struct {
	Provider string
	Bucket   string
	URL      string
}

type GetObjectResult struct {
	ContentType string
	Body        io.ReadCloser
}

type StorageProvider interface {
	ProviderName() string
	PutObject(ctx context.Context, input PutObjectInput) (*PutObjectResult, error)
	GetObject(ctx context.Context, objectKey string) (*GetObjectResult, error)
	DeleteObject(ctx context.Context, objectKey string) error
}

func newStorageProviderFromEnv() (StorageProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("MEDIA_STORAGE_PROVIDER")))
	if provider == "" {
		provider = "local"
	}

	switch provider {
	case "local":
		return newLocalStorageProviderFromEnv(), nil
	case "r2":
		return newS3CompatibleProviderFromEnv("r2")
	case "s3":
		return newS3CompatibleProviderFromEnv("s3")
	case "cos":
		return newS3CompatibleProviderFromEnv("cos")
	default:
		return nil, fmt.Errorf("unsupported MEDIA_STORAGE_PROVIDER: %s", provider)
	}
}

type s3CompatibleProvider struct {
	name          string
	endpoint      string
	bucket        string
	region        string
	accessKeyID   string
	secretKey     string
	publicBaseURL string
	httpClient    *http.Client
}

func newS3CompatibleProviderFromEnv(providerName string) (StorageProvider, error) {
	endpoint := firstNonEmpty(
		os.Getenv("MEDIA_S3_ENDPOINT"),
		os.Getenv("R2_ENDPOINT"),
	)
	bucket := firstNonEmpty(
		os.Getenv("MEDIA_S3_BUCKET"),
		os.Getenv("R2_BUCKET"),
	)
	region := firstNonEmpty(
		os.Getenv("MEDIA_S3_REGION"),
		os.Getenv("R2_REGION"),
		"auto",
	)
	accessKeyID := firstNonEmpty(
		os.Getenv("MEDIA_S3_ACCESS_KEY_ID"),
		os.Getenv("R2_ACCESS_KEY_ID"),
	)
	secretKey := firstNonEmpty(
		os.Getenv("MEDIA_S3_SECRET_ACCESS_KEY"),
		os.Getenv("R2_SECRET_ACCESS_KEY"),
	)
	publicBaseURL := firstNonEmpty(
		os.Getenv("MEDIA_S3_PUBLIC_BASE_URL"),
		os.Getenv("R2_PUBLIC_BASE_URL"),
	)

	if endpoint == "" || bucket == "" || accessKeyID == "" || secretKey == "" {
		return nil, errors.New("missing S3-compatible storage env: endpoint/bucket/access key/secret key")
	}

	normalizedEndpoint, err := normalizeS3Endpoint(endpoint, bucket)
	if err != nil {
		return nil, err
	}

	return &s3CompatibleProvider{
		name:          providerName,
		endpoint:      normalizedEndpoint,
		bucket:        strings.TrimSpace(bucket),
		region:        strings.TrimSpace(region),
		accessKeyID:   strings.TrimSpace(accessKeyID),
		secretKey:     strings.TrimSpace(secretKey),
		publicBaseURL: strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func normalizeS3Endpoint(rawEndpoint string, bucket string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawEndpoint))
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errors.New("invalid MEDIA_S3_ENDPOINT")
	}

	bucketPath := "/" + strings.Trim(strings.TrimSpace(bucket), "/")
	cleanPath := "/" + strings.Trim(strings.TrimSpace(u.Path), "/")
	if cleanPath == "/" {
		cleanPath = ""
	}
	// If user pasted a bucket-scoped endpoint, strip the bucket suffix once.
	if cleanPath == bucketPath {
		u.Path = ""
		u.RawPath = ""
	}

	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	return strings.TrimRight(u.String(), "/"), nil
}

func (p *s3CompatibleProvider) ProviderName() string {
	return p.name
}

func (p *s3CompatibleProvider) PutObject(ctx context.Context, input PutObjectInput) (*PutObjectResult, error) {
	if input.ObjectKey == "" {
		return nil, errors.New("object key is required")
	}

	bodyBytes, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}

	req, err := p.newSignedRequest(ctx, http.MethodPut, input.ObjectKey, bodyBytes, input.ContentType)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s3 put object failed: status=%d body=%s", resp.StatusCode, string(msg))
	}

	return &PutObjectResult{
		Provider: p.ProviderName(),
		Bucket:   p.bucket,
		URL:      p.objectPublicURL(input.ObjectKey),
	}, nil
}

func (p *s3CompatibleProvider) GetObject(ctx context.Context, objectKey string) (*GetObjectResult, error) {
	if objectKey == "" {
		return nil, errors.New("object key is required")
	}

	req, err := p.newSignedRequest(ctx, http.MethodGet, objectKey, nil, "")
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s3 get object failed: status=%d body=%s", resp.StatusCode, string(msg))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &GetObjectResult{
		ContentType: contentType,
		Body:        resp.Body,
	}, nil
}

func (p *s3CompatibleProvider) DeleteObject(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return errors.New("object key is required")
	}

	req, err := p.newSignedRequest(ctx, http.MethodDelete, objectKey, nil, "")
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete object failed: status=%d body=%s", resp.StatusCode, string(msg))
	}

	return nil
}

func (p *s3CompatibleProvider) newSignedRequest(ctx context.Context, method string, objectKey string, body []byte, contentType string) (*http.Request, error) {
	endpointURL, err := url.Parse(p.endpoint)
	if err != nil {
		return nil, err
	}

	basePath := strings.Trim(endpointURL.Path, "/")
	objectPath := encodePath(path.Join(p.bucket, objectKey))
	if basePath != "" {
		objectPath = encodePath(path.Join(basePath, p.bucket, objectKey))
	}
	canonicalURI := "/" + strings.TrimPrefix(objectPath, "/")
	requestURL := p.endpoint + canonicalURI

	payloadHash := calculatePayloadHash(method, body)
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	canonicalQuery := ""
	host := endpointURL.Host

	canonicalHeaders := "host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := method + "\n" +
		canonicalURI + "\n" +
		canonicalQuery + "\n" +
		canonicalHeaders + "\n" +
		signedHeaders + "\n" +
		payloadHash

	scope := dateStamp + "/" + p.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(signingKey(p.secretKey, dateStamp, p.region, "s3"), stringToSign))
	authorization := "AWS4-HMAC-SHA256 Credential=" + p.accessKeyID + "/" + scope + ", SignedHeaders=" + signedHeaders + ", Signature=" + signature

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("Authorization", authorization)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func (p *s3CompatibleProvider) objectPublicURL(objectKey string) string {
	if p.publicBaseURL != "" {
		return p.publicBaseURL + "/" + encodePath(objectKey)
	}
	return p.endpoint + "/" + p.bucket + "/" + encodePath(objectKey)
}

func signingKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func calculatePayloadHash(method string, body []byte) string {
	upperMethod := strings.ToUpper(strings.TrimSpace(method))
	if len(body) == 0 && (upperMethod == http.MethodGet || upperMethod == http.MethodHead || upperMethod == http.MethodDelete) {
		// For header-signed GET/HEAD/DELETE requests, S3-compatible services generally expect
		// the SHA256 hash of empty payload instead of UNSIGNED-PAYLOAD.
		return "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	}
	if len(body) == 0 {
		return "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	}
	return sha256Hex(body)
}

func canonicalizeQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		escapedKey := awsPercentEncode(k)
		for _, v := range vals {
			pairs = append(pairs, escapedKey+"="+awsPercentEncode(v))
		}
	}
	return strings.Join(pairs, "&")
}

func encodePath(p string) string {
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	for i, part := range parts {
		parts[i] = awsPercentEncode(part)
	}
	return strings.Join(parts, "/")
}

func awsPercentEncode(in string) string {
	encoded := url.QueryEscape(in)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

type localStorageProvider struct {
	rootDir string
	baseURL string
}

func newLocalStorageProviderFromEnv() StorageProvider {
	rootDir := strings.TrimSpace(os.Getenv("MEDIA_LOCAL_ROOT_DIR"))
	if rootDir == "" {
		rootDir = filepath.Join(os.TempDir(), "cyimewrite-media")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("MEDIA_LOCAL_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "/media-files"
	}
	return &localStorageProvider{
		rootDir: rootDir,
		baseURL: baseURL,
	}
}

func (p *localStorageProvider) ProviderName() string {
	return "local"
}

func (p *localStorageProvider) PutObject(_ context.Context, input PutObjectInput) (*PutObjectResult, error) {
	if input.ObjectKey == "" {
		return nil, errors.New("object key is required")
	}

	dstPath := filepath.Join(p.rootDir, filepath.FromSlash(input.ObjectKey))
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return nil, err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := io.Copy(f, input.Body); err != nil {
		return nil, err
	}

	url := p.baseURL + "/" + input.ObjectKey
	return &PutObjectResult{
		Provider: p.ProviderName(),
		Bucket:   "local",
		URL:      url,
	}, nil
}

func (p *localStorageProvider) GetObject(_ context.Context, objectKey string) (*GetObjectResult, error) {
	if objectKey == "" {
		return nil, errors.New("object key is required")
	}

	filePath := filepath.Join(p.rootDir, filepath.FromSlash(objectKey))
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(objectKey))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &GetObjectResult{
		ContentType: contentType,
		Body:        f,
	}, nil
}

func (p *localStorageProvider) DeleteObject(_ context.Context, objectKey string) error {
	if objectKey == "" {
		return errors.New("object key is required")
	}

	filePath := filepath.Join(p.rootDir, filepath.FromSlash(objectKey))
	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}
