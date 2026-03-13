package media

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultAssetGCInterval = time.Minute
	defaultAssetGCBatch    = 20
)

func StartAssetGCWorker(ctx context.Context) {
	if !assetGCEnabledFromEnv() {
		log.Println("[media.gc] disabled")
		return
	}

	interval := assetGCIntervalFromEnv()
	batchSize := assetGCBatchSizeFromEnv()
	if interval <= 0 || batchSize <= 0 {
		log.Printf("[media.gc] skipped invalid config interval=%s batch=%d", interval, batchSize)
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Printf("[media.gc] started interval=%s batch=%d", interval, batchSize)
		for {
			if _, err := RunDueAssetGCJobs(ctx, time.Now(), batchSize); err != nil {
				log.Printf("[media.gc] run failed: %v", err)
			}

			select {
			case <-ctx.Done():
				log.Println("[media.gc] stopped")
				return
			case <-ticker.C:
			}
		}
	}()
}

func RunDueAssetGCJobs(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = defaultAssetGCBatch
	}

	var jobs []models.AssetGCJob
	if err := database.DB.
		Where("job_type = ? AND status = ? AND run_after <= ?", "delete", "pending", now).
		Order("run_after asc").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return 0, err
	}

	processed := 0
	for _, job := range jobs {
		processed++
		if err := runAssetDeleteJob(ctx, job.ID, now); err != nil {
			log.Printf("[media.gc] job=%s failed: %v", job.ID, err)
		}
	}

	return processed, nil
}

func runAssetDeleteJob(ctx context.Context, jobID uuid.UUID, now time.Time) error {
	var job models.AssetGCJob
	var asset models.Asset
	var shouldDelete bool

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND job_type = ? AND status = ?", jobID, "delete", "pending").First(&job).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.AssetGCJob{}).
			Where("id = ?", job.ID).
			Updates(map[string]any{
				"status":        "running",
				"attempt_count": gorm.Expr("attempt_count + 1"),
				"updated_at":    now,
				"last_error":    nil,
			}).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ? AND deleted_at IS NULL", job.AssetID).First(&asset).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tx.Model(&models.AssetGCJob{}).
					Where("id = ?", job.ID).
					Updates(map[string]any{
						"status":     "done",
						"updated_at": now,
					}).Error
			}
			return err
		}

		var refCount int64
		if err := tx.Model(&models.DocumentAssetRef{}).
			Where("asset_id = ? AND ref_type = ?", job.AssetID, "editor_content").
			Count(&refCount).Error; err != nil {
			return err
		}
		if refCount > 0 {
			if err := tx.Model(&models.Asset{}).
				Where("id = ?", asset.ID).
				Updates(map[string]any{
					"status":          "ready",
					"reference_count": int(refCount),
					"updated_at":      now,
				}).Error; err != nil {
				return err
			}
			return tx.Model(&models.AssetGCJob{}).
				Where("id = ?", job.ID).
				Updates(map[string]any{
					"status":     "cancelled",
					"updated_at": now,
				}).Error
		}

		shouldDelete = true
		return nil
	})
	if err != nil {
		return markAssetGCJobFailed(jobID, now, err)
	}
	if asset.ID == uuid.Nil || !shouldDelete {
		return nil
	}

	if err := initStorageProvider(); err != nil {
		return markAssetGCJobFailed(jobID, now, err)
	}
	if err := storageProvider.DeleteObject(ctx, asset.ObjectKey); err != nil {
		return markAssetGCJobFailed(jobID, now, err)
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Asset{}).
			Where("id = ? AND deleted_at IS NULL", asset.ID).
			Updates(map[string]any{
				"status":          "deleted",
				"reference_count": 0,
				"updated_at":      now,
				"deleted_at":      now,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&models.AssetGCJob{}).
			Where("id = ?", jobID).
			Updates(map[string]any{
				"status":     "done",
				"updated_at": now,
			}).Error
	})
}

func markAssetGCJobFailed(jobID uuid.UUID, now time.Time, cause error) error {
	message := cause.Error()
	updateErr := database.DB.Model(&models.AssetGCJob{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":     "failed",
			"last_error": message,
			"updated_at": now,
		}).Error
	if updateErr != nil {
		return errors.Join(cause, updateErr)
	}
	return cause
}

func assetGCEnabledFromEnv() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("MEDIA_ASSET_GC_ENABLED")))
	return raw == "" || raw == "1" || raw == "true" || raw == "yes"
}

func assetGCIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("MEDIA_ASSET_GC_INTERVAL"))
	if raw == "" {
		return defaultAssetGCInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultAssetGCInterval
	}
	return d
}

func assetGCBatchSizeFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("MEDIA_ASSET_GC_BATCH_SIZE"))
	if raw == "" {
		return defaultAssetGCBatch
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultAssetGCBatch
	}
	return n
}
