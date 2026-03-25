package media

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
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

func TestUploadDocumentImage_UsesSeeForUserConfigTargets(t *testing.T) {
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
		APIToken:     stringPtr("test-see-token"),
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"url": "https://s.ee/example.png",
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
		APIToken:     stringPtr("test-lsky-token"),
		ConfigJSON:   stringPtr(`{"storageId":7,"strategyId":"covers"}`),
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

func stringPtr(value string) *string {
	return &value
}
