package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
)

type mockStorageProvider struct {
	deleteCalls []string
	deleteErr   error
}

func (m *mockStorageProvider) ProviderName() string {
	return "mock"
}

func (m *mockStorageProvider) PutObject(_ context.Context, _ PutObjectInput) (*PutObjectResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorageProvider) GetObject(_ context.Context, _ string) (*GetObjectResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorageProvider) DeleteObject(_ context.Context, objectKey string) error {
	m.deleteCalls = append(m.deleteCalls, objectKey)
	return m.deleteErr
}

func TestRunDueAssetGCJobs_DeletesUnusedAssets(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)
	now := time.Now()

	mock := &mockStorageProvider{}
	storageProvider = mock
	t.Cleanup(func() { storageProvider = nil })

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "unused.png",
		FileHash:        "hash-unused",
		FileSize:        3,
		MimeType:        "image/png",
		StorageProvider: "mock",
		ObjectKey:       "owner/unused.png",
		URL:             "http://example.test/unused.png",
		Visibility:      "private",
		Status:          "pending_delete",
		ReferenceCount:  0,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	job := models.AssetGCJob{
		ID:       uuid.New(),
		AssetID:  asset.ID,
		JobType:  "delete",
		Status:   "pending",
		RunAfter: now.Add(-time.Minute),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	processed, err := RunDueAssetGCJobs(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("run gc jobs: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed=1, got %d", processed)
	}
	if len(mock.deleteCalls) != 1 || mock.deleteCalls[0] != asset.ObjectKey {
		t.Fatalf("unexpected delete calls: %+v", mock.deleteCalls)
	}

	var gotAsset models.Asset
	if err := db.Unscoped().First(&gotAsset, "id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load asset: %v", err)
	}
	if gotAsset.Status != "deleted" || !gotAsset.DeletedAt.Valid {
		t.Fatalf("expected deleted asset, got status=%s deleted=%v", gotAsset.Status, gotAsset.DeletedAt.Valid)
	}

	var gotJob models.AssetGCJob
	if err := db.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if gotJob.Status != "done" || gotJob.AttemptCount != 1 {
		t.Fatalf("expected done job with attempt_count=1, got status=%s attempts=%d", gotJob.Status, gotJob.AttemptCount)
	}
}

func TestRunDueAssetGCJobs_CancelsWhenAssetIsReferencedAgain(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)
	now := time.Now()

	mock := &mockStorageProvider{}
	storageProvider = mock
	t.Cleanup(func() { storageProvider = nil })

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "used.png",
		FileHash:        "hash-used-worker",
		FileSize:        3,
		MimeType:        "image/png",
		StorageProvider: "mock",
		ObjectKey:       "owner/used.png",
		URL:             "http://example.test/used.png",
		Visibility:      "private",
		Status:          "pending_delete",
		ReferenceCount:  0,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Create(&models.DocumentAssetRef{
		ID:          uuid.New(),
		DocumentID:  docID,
		AssetID:     asset.ID,
		OwnerUserID: userID,
		RefType:     "editor_content",
	}).Error; err != nil {
		t.Fatalf("create ref: %v", err)
	}
	job := models.AssetGCJob{
		ID:       uuid.New(),
		AssetID:  asset.ID,
		JobType:  "delete",
		Status:   "pending",
		RunAfter: now.Add(-time.Minute),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	processed, err := RunDueAssetGCJobs(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("run gc jobs: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed=1, got %d", processed)
	}
	if len(mock.deleteCalls) != 0 {
		t.Fatalf("expected no delete calls, got %+v", mock.deleteCalls)
	}

	var gotAsset models.Asset
	if err := db.First(&gotAsset, "id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load asset: %v", err)
	}
	if gotAsset.Status != "ready" || gotAsset.ReferenceCount != 1 {
		t.Fatalf("expected ready asset with ref=1, got status=%s ref=%d", gotAsset.Status, gotAsset.ReferenceCount)
	}

	var gotJob models.AssetGCJob
	if err := db.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if gotJob.Status != "cancelled" || gotJob.AttemptCount != 1 {
		t.Fatalf("expected cancelled job with attempt_count=1, got status=%s attempts=%d", gotJob.Status, gotJob.AttemptCount)
	}
}

func TestRunDueAssetGCJobs_MarksFailedOnDeleteError(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)
	now := time.Now()

	mock := &mockStorageProvider{deleteErr: errors.New("boom")}
	storageProvider = mock
	t.Cleanup(func() { storageProvider = nil })

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "broken.png",
		FileHash:        "hash-broken",
		FileSize:        3,
		MimeType:        "image/png",
		StorageProvider: "mock",
		ObjectKey:       "owner/broken.png",
		URL:             "http://example.test/broken.png",
		Visibility:      "private",
		Status:          "pending_delete",
		ReferenceCount:  0,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	job := models.AssetGCJob{
		ID:       uuid.New(),
		AssetID:  asset.ID,
		JobType:  "delete",
		Status:   "pending",
		RunAfter: now.Add(-time.Minute),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	processed, err := RunDueAssetGCJobs(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("run gc jobs: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed=1, got %d", processed)
	}

	var gotAsset models.Asset
	if err := db.First(&gotAsset, "id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load asset: %v", err)
	}
	if gotAsset.Status != "pending_delete" || gotAsset.DeletedAt.Valid {
		t.Fatalf("expected asset to stay pending_delete, got status=%s deleted=%v", gotAsset.Status, gotAsset.DeletedAt.Valid)
	}

	var gotJob models.AssetGCJob
	if err := db.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if gotJob.Status != "pending" || gotJob.AttemptCount != 1 {
		t.Fatalf("expected pending retry job with attempt_count=1, got status=%s attempts=%d", gotJob.Status, gotJob.AttemptCount)
	}
	if gotJob.LastError == nil || *gotJob.LastError != "boom" {
		t.Fatalf("expected last_error=boom, got %+v", gotJob.LastError)
	}
	if !gotJob.RunAfter.After(now) {
		t.Fatalf("expected retry run_after to be pushed forward, got %s", gotJob.RunAfter)
	}
}

func TestRunDueAssetGCJobs_MarksFailedWhenMaxAttemptsReached(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)
	now := time.Now()
	t.Setenv("MEDIA_ASSET_GC_MAX_ATTEMPTS", "1")

	mock := &mockStorageProvider{deleteErr: errors.New("boom-final")}
	storageProvider = mock
	t.Cleanup(func() { storageProvider = nil })

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "broken-final.png",
		FileHash:        "hash-broken-final",
		FileSize:        3,
		MimeType:        "image/png",
		StorageProvider: "mock",
		ObjectKey:       "owner/broken-final.png",
		URL:             "http://example.test/broken-final.png",
		Visibility:      "private",
		Status:          "pending_delete",
		ReferenceCount:  0,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	job := models.AssetGCJob{
		ID:       uuid.New(),
		AssetID:  asset.ID,
		JobType:  "delete",
		Status:   "pending",
		RunAfter: now.Add(-time.Minute),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	processed, err := RunDueAssetGCJobs(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("run gc jobs: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed=1, got %d", processed)
	}

	var gotJob models.AssetGCJob
	if err := db.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if gotJob.Status != "failed" || gotJob.AttemptCount != 1 {
		t.Fatalf("expected final failed job with attempt_count=1, got status=%s attempts=%d", gotJob.Status, gotJob.AttemptCount)
	}
	if gotJob.LastError == nil || *gotJob.LastError != "boom-final" {
		t.Fatalf("expected last_error=boom-final, got %+v", gotJob.LastError)
	}
}

func TestRunAssetReferenceReconcilePass_RepairsDriftedAssetState(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)
	now := time.Now()

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "drifted.png",
		FileHash:        "hash-drifted",
		FileSize:        3,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       "owner/drifted.png",
		URL:             "http://example.test/drifted.png",
		Visibility:      "private",
		Status:          "ready",
		ReferenceCount:  99,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}

	reconciled, err := RunAssetReferenceReconcilePass(now, 10)
	if err != nil {
		t.Fatalf("reconcile pass: %v", err)
	}
	if reconciled != 1 {
		t.Fatalf("expected reconciled=1, got %d", reconciled)
	}

	var gotAsset models.Asset
	if err := db.First(&gotAsset, "id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load asset: %v", err)
	}
	if gotAsset.ReferenceCount != 0 || gotAsset.Status != "pending_delete" {
		t.Fatalf("expected asset repaired to pending_delete with ref=0, got status=%s ref=%d", gotAsset.Status, gotAsset.ReferenceCount)
	}

	var pendingJobs []models.AssetGCJob
	if err := db.Where("asset_id = ? AND status = ?", asset.ID, "pending").Find(&pendingJobs).Error; err != nil {
		t.Fatalf("load pending jobs: %v", err)
	}
	if len(pendingJobs) != 1 || pendingJobs[0].JobType != "delete" {
		t.Fatalf("expected one pending delete job, got %+v", pendingJobs)
	}
}
