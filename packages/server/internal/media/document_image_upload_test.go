package media

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"g.co1d.in/Coldin04/Cyime/server/internal/models"
	"g.co1d.in/Coldin04/Cyime/server/internal/securevalue"
	"github.com/google/uuid"
)

func TestUploadDocumentImage_UsesManagedR2ForLegacyDocuments(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	if err := db.Create(&models.User{ID: userID}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	docID := seedOwnedDocumentWithImageTarget(t, db, userID, "")
	if err := db.Model(&models.Document{}).
		Where("id = ?", docID).
		Update("preferred_image_target_id", "").Error; err != nil {
		t.Fatalf("force legacy image target: %v", err)
	}

	t.Setenv("MEDIA_STORAGE_PROVIDER", "local")
	t.Setenv("MEDIA_LOCAL_ROOT_DIR", t.TempDir())
	storageProvider = nil
	t.Cleanup(func() { storageProvider = nil })

	header := makeFileHeader(t, "file", "legacy.png", []byte("managed"))
	result, err := UploadDocumentImage(context.Background(), UploadDocumentImageRequest{
		DocumentID: docID,
		UserID:     userID,
		FileHeader: header,
	})
	if err != nil {
		t.Fatalf("upload document image: %v", err)
	}

	if result.TargetID != documentImageTargetManagedR2 {
		t.Fatalf("expected managed-r2 target, got %s", result.TargetID)
	}
	if result.Mode != documentImageModeManagedAsset {
		t.Fatalf("expected managed asset mode, got %s", result.Mode)
	}
	if result.AssetID == nil {
		t.Fatalf("expected asset id for managed upload")
	}
}

func TestUploadDocumentImage_FallsBackToManagedR2WhenSharedTargetBelongsToAnotherUser(t *testing.T) {
	db := setupMediaTestDB(t)
	ownerID := uuid.New()
	editorID := uuid.New()
	if err := db.Create(&models.User{ID: ownerID}).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := db.Create(&models.User{ID: editorID}).Error; err != nil {
		t.Fatalf("create editor: %v", err)
	}

	ownerConfig := models.UserImageBedConfig{
		ID:           uuid.New(),
		UserID:       ownerID,
		Name:         "owner bed",
		ProviderType: "see",
		APIToken:     mustEncryptToken(t, "owner-token"),
		IsEnabled:    true,
	}
	if err := db.Create(&ownerConfig).Error; err != nil {
		t.Fatalf("create owner config: %v", err)
	}
	docID := seedOwnedDocumentWithImageTarget(t, db, ownerID, ownerConfig.ID.String())
	seedDocumentPermission(t, db, docID, editorID, ownerID, "editor")

	t.Setenv("MEDIA_STORAGE_PROVIDER", "local")
	t.Setenv("MEDIA_LOCAL_ROOT_DIR", t.TempDir())
	storageProvider = nil
	t.Cleanup(func() { storageProvider = nil })

	header := makeFileHeader(t, "file", "shared.png", []byte("managed-fallback"))
	result, err := UploadDocumentImage(context.Background(), UploadDocumentImageRequest{
		DocumentID: docID,
		UserID:     editorID,
		FileHeader: header,
	})
	if err != nil {
		t.Fatalf("upload document image: %v", err)
	}

	if result.TargetID != documentImageTargetManagedR2 {
		t.Fatalf("expected managed-r2 fallback target, got %s", result.TargetID)
	}
	if result.Mode != documentImageModeManagedAsset {
		t.Fatalf("expected managed asset mode, got %s", result.Mode)
	}
	if result.AssetID == nil {
		t.Fatalf("expected managed fallback to return asset id")
	}
}

func TestUploadDocumentImage_UsesSeeForUserConfigTargets(t *testing.T) {
	allowInsecureImageBedDialsForTesting(t)
	db := setupMediaTestDB(t)
	userID := uuid.New()
	if err := db.Create(&models.User{ID: userID}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	config := models.UserImageBedConfig{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         "see",
		ProviderType: "see",
		APIToken:     mustEncryptToken(t, "test-see-token"),
		IsEnabled:    true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("create image bed config: %v", err)
	}
	docID := seedOwnedDocumentWithImageTarget(t, db, userID, config.ID.String())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "test-see-token" {
			t.Fatalf("expected authorization header, got %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Fatalf("expected multipart form upload, got %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		fileHeaders := r.MultipartForm.File["file"]
		if len(fileHeaders) != 1 {
			t.Fatalf("expected one file part, got %d", len(fileHeaders))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 200,
			"data": map[string]any{
				"upload_status": 1,
				"url":           "https://s.ee/example.png",
			},
		})
	}))
	defer server.Close()

	t.Setenv("SEE_API_BASE_URL", server.URL)

	header := makeFileHeader(t, "file", "public.png", []byte("public"))
	result, err := UploadDocumentImage(context.Background(), UploadDocumentImageRequest{
		DocumentID: docID,
		UserID:     userID,
		FileHeader: header,
	})
	if err != nil {
		t.Fatalf("upload document image: %v", err)
	}

	if result.TargetID != config.ID.String() {
		t.Fatalf("expected config target, got %s", result.TargetID)
	}
	if result.Mode != documentImageModeExternalURL {
		t.Fatalf("expected external url mode, got %s", result.Mode)
	}
	if result.URL != "https://s.ee/example.png" {
		t.Fatalf("unexpected url: %s", result.URL)
	}
	if result.AssetID != nil {
		t.Fatalf("expected no asset id for see upload")
	}
}

func TestUploadDocumentImage_UsesLskyForUserConfigTargets(t *testing.T) {
	allowInsecureImageBedDialsForTesting(t)
	db := setupMediaTestDB(t)
	userID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/upload" {
			t.Fatalf("expected /api/v2/upload, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-lsky-token" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected accept header, got %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Fatalf("expected multipart form upload, got %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if got := r.FormValue("storage_id"); got != "7" {
			t.Fatalf("expected storage_id=7, got %q", got)
		}
		if got := r.FormValue("strategy_id"); got != "covers" {
			t.Fatalf("expected strategy_id=covers, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"public_url": "https://cdn.example.test/demo.png",
			},
		})
	}))
	defer server.Close()

	if err := db.Create(&models.User{ID: userID}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	config := models.UserImageBedConfig{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         "lsky",
		ProviderType: "lsky",
		BaseURL:      stringPtr(server.URL),
		APIToken:     mustEncryptToken(t, "test-lsky-token"),
		ConfigJSON:   stringPtr(`{"fields":{"storageId":7,"strategyId":"covers"}}`),
		IsEnabled:    true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("create image bed config: %v", err)
	}
	docID := seedOwnedDocumentWithImageTarget(t, db, userID, config.ID.String())

	header := makeFileHeader(t, "file", "public.png", []byte("public"))
	result, err := UploadDocumentImage(context.Background(), UploadDocumentImageRequest{
		DocumentID: docID,
		UserID:     userID,
		FileHeader: header,
	})
	if err != nil {
		t.Fatalf("upload document image: %v", err)
	}

	if result.TargetID != config.ID.String() {
		t.Fatalf("expected config target, got %s", result.TargetID)
	}
	if result.Mode != documentImageModeExternalURL {
		t.Fatalf("expected external url mode, got %s", result.Mode)
	}
	if result.URL != "https://cdn.example.test/demo.png" {
		t.Fatalf("unexpected normalized url: %s", result.URL)
	}
}

func TestUploadDocumentImage_UsesImgBBForUserConfigTargets(t *testing.T) {
	allowInsecureImageBedDialsForTesting(t)
	db := setupMediaTestDB(t)
	userID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/1/upload" {
			t.Fatalf("expected /1/upload, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("key"); got != "imgbb-key" {
			t.Fatalf("expected key=imgbb-key, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected accept header, got %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Fatalf("expected multipart form upload, got %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		fileHeaders := r.MultipartForm.File["image"]
		if len(fileHeaders) != 1 {
			t.Fatalf("expected one image part, got %d", len(fileHeaders))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"status":  200,
			"data": map[string]any{
				"url": "https://i.ibb.co/demo/demo.png",
			},
		})
	}))
	defer server.Close()

	if err := db.Create(&models.User{ID: userID}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	config := models.UserImageBedConfig{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         "imgbb",
		ProviderType: "imgbb",
		APIToken:     mustEncryptToken(t, "imgbb-key"),
		IsEnabled:    true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("create image bed config: %v", err)
	}
	docID := seedOwnedDocumentWithImageTarget(t, db, userID, config.ID.String())
	t.Setenv("IMGBB_API_BASE_URL", server.URL+"/1")

	header := makeFileHeader(t, "file", "public.png", []byte("imgbb"))
	result, err := UploadDocumentImage(context.Background(), UploadDocumentImageRequest{
		DocumentID: docID,
		UserID:     userID,
		FileHeader: header,
	})
	if err != nil {
		t.Fatalf("upload document image: %v", err)
	}

	if result.TargetID != config.ID.String() {
		t.Fatalf("expected config target, got %s", result.TargetID)
	}
	if result.Mode != documentImageModeExternalURL {
		t.Fatalf("expected external url mode, got %s", result.Mode)
	}
	if result.URL != "https://i.ibb.co/demo/demo.png" {
		t.Fatalf("unexpected normalized url: %s", result.URL)
	}
}

func TestUploadDocumentImage_UsesCheveretoForUserConfigTargets(t *testing.T) {
	allowInsecureImageBedDialsForTesting(t)
	db := setupMediaTestDB(t)
	userID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/1/upload" {
			t.Fatalf("expected /api/1/upload, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("key"); got != "chevereto-key" {
			t.Fatalf("expected key=chevereto-key, got %q", got)
		}
		if got := r.URL.Query().Get("format"); got != "json" {
			t.Fatalf("expected format=json, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected accept header, got %q", got)
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		fileHeaders := r.MultipartForm.File["source"]
		if len(fileHeaders) != 1 {
			t.Fatalf("expected one source part, got %d", len(fileHeaders))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status_code": 200,
			"image": map[string]any{
				"url": "https://img.example.test/uploads/chevereto-demo.png",
			},
		})
	}))
	defer server.Close()

	if err := db.Create(&models.User{ID: userID}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	config := models.UserImageBedConfig{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         "chevereto",
		ProviderType: "chevereto",
		BaseURL:      stringPtr(server.URL),
		APIToken:     mustEncryptToken(t, "chevereto-key"),
		IsEnabled:    true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("create image bed config: %v", err)
	}
	docID := seedOwnedDocumentWithImageTarget(t, db, userID, config.ID.String())

	header := makeFileHeader(t, "file", "public.png", []byte("chevereto"))
	result, err := UploadDocumentImage(context.Background(), UploadDocumentImageRequest{
		DocumentID: docID,
		UserID:     userID,
		FileHeader: header,
	})
	if err != nil {
		t.Fatalf("upload document image: %v", err)
	}

	if result.TargetID != config.ID.String() {
		t.Fatalf("expected config target, got %s", result.TargetID)
	}
	if result.Mode != documentImageModeExternalURL {
		t.Fatalf("expected external url mode, got %s", result.Mode)
	}
	if result.URL != "https://img.example.test/uploads/chevereto-demo.png" {
		t.Fatalf("unexpected normalized url: %s", result.URL)
	}
}

func stringPtr(value string) *string {
	return &value
}

func mustEncryptToken(t *testing.T, value string) *string {
	t.Helper()
	encrypted, err := securevalue.EncryptString(value)
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	return &encrypted
}
