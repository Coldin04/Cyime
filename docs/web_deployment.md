# Web 部署说明

当前前端只部署 `packages/web`，后端与 realtime 独立部署。

## 通用设置

- 仓库根目录：`/`
- 前端项目目录：`packages/web`
- Node 版本：`20`
- 构建期公开环境变量：`PUBLIC_API_BASE_URL`
- 可选公开环境变量：`PUBLIC_AVATAR_MAX_BYTES`、`PUBLIC_AVATAR_OUTPUT_SIZE`

## Cloudflare Pages

- Framework preset：`SvelteKit`
- Root directory：`packages/web`
- Build command：`pnpm install --frozen-lockfile && pnpm run build:cloudflare`
- Build output directory：`.svelte-kit/cloudflare`
- 仓库内已提供 `packages/web/wrangler.toml`，默认包含：
  - `name = "cyimewrite-web"`
  - `pages_build_output_dir = ".svelte-kit/cloudflare"`
  - `compatibility_flags = ["nodejs_compat"]`

推荐同时配置：

- `PUBLIC_API_BASE_URL=https://你的后端域名`
- `PUBLIC_AVATAR_MAX_BYTES=2097152`
- `PUBLIC_AVATAR_OUTPUT_SIZE=512`

## EdgeOne Pages

- Root directory：`packages/web`
- Install command：`pnpm install --frozen-lockfile`
- Build command：`pnpm run build:edgeone`
- Output directory：`.edgeone/output`

推荐同时配置：

- `PUBLIC_API_BASE_URL=https://你的后端域名`
- `PUBLIC_AVATAR_MAX_BYTES=2097152`
- `PUBLIC_AVATAR_OUTPUT_SIZE=512`

## 本地验证

在 `packages/web` 目录下执行：

```bash
pnpm run build:cloudflare
pnpm run build:edgeone
```
