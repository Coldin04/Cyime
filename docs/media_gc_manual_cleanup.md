# 媒体库手动回收与删除（`media_gc`）

本文说明如何在后端手动触发一次全局媒体回收，并在需要时立即向 R2 发起物理删除请求。

## 1. 命令入口

在 `packages/server` 目录执行：

```bash
go run ./cmd/media_gc
```

## 2. 默认行为（安全模式）

```bash
go run ./cmd/media_gc -immediate=false
```

默认建议先用这类模式做巡检：
- 扫描全局 `assets` 引用状态（对账）
- 为未引用资源创建/维护删除任务
- 不强制把历史 `pending` 任务提前到当前时刻

## 3. 立即回收（会推进到物理删除）

```bash
go run ./cmd/media_gc -immediate=true -run-due-gc=true
```

当 `-immediate=true` 时，命令会：
- 将 `MEDIA_ASSET_DELETE_DELAY=0s`
- 将 `MEDIA_BLOB_DELETE_DELAY=0s`
- 把历史 `pending` 的 `asset_gc_jobs`/`blob_gc_jobs` 的 `run_after` 重排为当前时间
- 在本次执行中直接处理到期 GC 任务

如果目标 `blob` 已无任何活跃 `asset` 引用，后端会调用存储 provider（R2/S3 兼容）删除对象。

## 4. 常用参数

- `-reconcile-batch`：每轮对账扫描数量，默认 `500`
- `-reconcile-loops`：最多循环轮数，默认 `50`
- `-gc-batch`：每轮处理 GC 任务数量，默认 `200`
- `-immediate`：是否立即重排/执行删除任务，默认 `true`
- `-run-due-gc`：是否执行到期 GC，默认 `true`

## 5. 日志解读

- `reconciled=N`：本次对账扫描到的资产数量
- `due gc processed asset_jobs=A blob_jobs=B`：
  - `A` 表示执行了多少资产删除任务
  - `B` 表示执行了多少物理对象删除任务（会触发 R2 删除请求）

示例：
- `asset_jobs=2 blob_jobs=0`：说明资产层已处理，但当前没有可执行的 blob 删除任务（或仍有引用）

## 6. 典型排查

查看是否还有到期任务可执行：

```sql
select count(*) from asset_gc_jobs where status='pending' and run_after <= datetime('now');
select count(*) from blob_gc_jobs where status='pending' and run_after <= datetime('now');
```

查看 blob 是否仍被活跃资产引用：

```sql
select blob_id, count(*) as active_assets
from assets
where deleted_at is null
group by blob_id;
```

