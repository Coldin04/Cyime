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
)

func main() {
	_ = config.LoadDotEnv(".env")

	var (
		reconcileBatch = flag.Int("reconcile-batch", 500, "每轮对账处理的 assets 数量")
		reconcileLoops = flag.Int("reconcile-loops", 50, "最多跑多少轮对账（防止无限循环）")
		gcBatch        = flag.Int("gc-batch", 200, "每轮处理的 GC job 数量")
		immediate      = flag.Bool("immediate", true, "是否创建“立即可执行”的删除任务（asset_gc_jobs.run_after=now）")
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
		// Let reconcileOneAsset enqueue jobs with run_after=now.
		_ = os.Setenv("MEDIA_ASSET_DELETE_DELAY", "0s")
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
