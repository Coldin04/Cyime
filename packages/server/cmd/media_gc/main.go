package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/config"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/media"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
)

func main() {
	_ = config.LoadDotEnv(".env")

	var (
		reconcileBatch = flag.Int("reconcile-batch", 500, "每轮对账处理的 assets 数量")
		reconcileLoops = flag.Int("reconcile-loops", 50, "最多跑多少轮对账（防止无限循环）")
		gcBatch        = flag.Int("gc-batch", 200, "每轮处理的 GC job 数量")
		immediate      = flag.Bool("immediate", false, "是否创建并重排“立即可执行”的删除任务（asset/blob gc run_after=now）")
		runDueGC       = flag.Bool("run-due-gc", false, "是否在本次命令中直接处理已到期的 GC job（asset + blob）")
	)
	flag.Parse()

	if *reconcileBatch <= 0 || *reconcileLoops <= 0 || *gcBatch <= 0 {
		_, _ = fmt.Fprintln(os.Stderr, "invalid flags: batch/loops must be > 0")
		os.Exit(2)
	}

	database.Connect()
	log.Println("[media-gc] database ready")

	ctx := context.Background()

	if *immediate {
		// Let reconcile/GC schedule both asset and blob deletions immediately.
		if err := os.Setenv("MEDIA_ASSET_DELETE_DELAY", "0s"); err != nil {
			log.Fatalf("[media-gc] set MEDIA_ASSET_DELETE_DELAY failed: %v", err)
		}
		if err := os.Setenv("MEDIA_BLOB_DELETE_DELAY", "0s"); err != nil {
			log.Fatalf("[media-gc] set MEDIA_BLOB_DELETE_DELAY failed: %v", err)
		}

		now := time.Now()
		assetRes := database.DB.Model(&models.AssetGCJob{}).
			Where("job_type = ? AND status = ? AND run_after > ?", "delete", "pending", now).
			Updates(map[string]any{
				"run_after":  now,
				"updated_at": now,
			})
		if assetRes.Error != nil {
			log.Fatalf("[media-gc] reschedule pending asset gc jobs failed: %v", assetRes.Error)
		}

		blobRes := database.DB.Model(&models.BlobGCJob{}).
			Where("job_type = ? AND status = ? AND run_after > ?", "delete", "pending", now).
			Updates(map[string]any{
				"run_after":  now,
				"updated_at": now,
			})
		if blobRes.Error != nil {
			log.Fatalf("[media-gc] reschedule pending blob gc jobs failed: %v", blobRes.Error)
		}
		log.Printf("[media-gc] immediate rescheduled pending jobs asset=%d blob=%d", assetRes.RowsAffected, blobRes.RowsAffected)
	}

	totalReconciled := 0
	for i := 0; i < *reconcileLoops; i++ {
		now := time.Now()
		n, err := media.RunAssetReferenceReconcilePass(now, *reconcileBatch)
		if err != nil {
			log.Fatalf("[media-gc] reconcile failed: %v", err)
		}
		totalReconciled += n
		log.Printf("[media-gc] reconcile pass=%d reconciled=%d total=%d", i+1, n, totalReconciled)
		if n < *reconcileBatch {
			break
		}
	}

	if *runDueGC {
		now := time.Now()
		assetJobs, err := media.RunDueAssetGCJobs(ctx, now, *gcBatch)
		if err != nil {
			log.Fatalf("[media-gc] run due asset gc jobs failed: %v", err)
		}
		blobJobs, err := media.RunDueBlobGCJobs(ctx, now, *gcBatch)
		if err != nil {
			log.Fatalf("[media-gc] run due blob gc jobs failed: %v", err)
		}
		log.Printf("[media-gc] due gc processed asset_jobs=%d blob_jobs=%d", assetJobs, blobJobs)
	}

	log.Printf("[media-gc] done reconciled=%d", totalReconciled)
}
