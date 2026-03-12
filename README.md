# CyimeWrite

🍋 CyimeWrite —— 青柠写  轻量简洁的云端同步Markdown编辑器 为随心流动的创意而生

---

## 开发指南 (Development Guide)

### 项目架构 (Project Architecture)

以下是本项目核心功能（特别是认证系统）相关的主要文件架构。

#### **后端 (`/packages/server`)**

-   `cmd/server/main.go`: **应用入口**。负责注册所有 API 路由并应用中间件。
-   `internal/auth/handler.go`: **认证处理器**。处理 `/login`, `/callback`, `/refresh` 等路由的主逻辑。
-   `internal/auth/token.go`: **令牌服务核心**。封装了所有关于令牌的创建、持久化和刷新逻辑。
-   `internal/middleware/auth.go`: **JWT 认证中间件**。提供 `Protected()` 中间件来保护需要认证的接口。
-   `internal/user/handler.go`: **用户处理器**。处理与用户数据相关的请求 (`/user/me`)。
-   `internal/database/database.go`: **数据库**。初始化 GORM 连接并执行自动迁移。
-   `internal/models/*.go`: **数据库模型**。定义了 `users`, `auth_providers` 等数据表结构。

#### **前端 (`/packages/web`)**

-   `src/lib/stores/auth.ts`: **前端认证的大脑**。通过 Svelte Store 集中管理认证状态和所有刷新逻辑。
-   `src/lib/api.ts`: **API 请求工具**。导出的 `apiFetch` 函数封装了原生 `fetch`，自动处理认证头和 401 错误重试。
-   `src/routes/auth/callback/+page.svelte`: **登录回调页**。处理从第三方登录成功后的跳转。
-   `src/routes/workspace/+layout.svelte`: **工作区路由守卫**。保护 `/workspace` 目录下的所有页面。

### 本地开发 (Local Development)

1.  **环境准备**:
    -   确保您已安装 Go (1.22+)。
    -   确保您已安装 Node.js (18+) 和 `pnpm`。

2.  **启动后端服务**:
    ```bash
    # 进入后端目录
    cd packages/server

    # (可选) 在该目录下创建一个 .env 文件来配置环境变量，例如:
    # JWT_SECRET_KEY=a-very-secret-key-for-development
    # CORS_ALLOWED_ORIGINS=http://localhost:5173

    # 运行后端
    go run ./cmd/server/main.go
    ```
    后端服务将运行在 `http://localhost:8080`。
    说明：后端启动会自动读取 `packages/server/.env`。

    媒体模块环境变量：
    - `MEDIA_TOKEN_SECRET=replace-with-strong-secret`（私有媒体签名必需）
    - `MEDIA_SIGN_TTL_SECONDS=120`
    - `MEDIA_STORAGE_PROVIDER=local|r2|s3|cos`（默认 `local`）
    - `RESET_WORKSPACE_TABLES_ON_BOOT=false`（默认不清空业务表；仅调试时改为 `true`）

    本地 provider（开发）：
    - `MEDIA_LOCAL_ROOT_DIR=/tmp/cyimewrite-media`
    - `MEDIA_LOCAL_BASE_URL=/media-files`

    S3 兼容 provider（R2/COS/S3）：
    - `MEDIA_S3_ENDPOINT=https://<endpoint>`
    - `MEDIA_S3_BUCKET=<bucket>`
    - `MEDIA_S3_REGION=auto`
    - `MEDIA_S3_ACCESS_KEY_ID=<key>`
    - `MEDIA_S3_SECRET_ACCESS_KEY=<secret>`
    - `MEDIA_S3_PUBLIC_BASE_URL=https://<cdn-domain>`（可选）

    R2 也可用同义变量（兼容）：
    - `R2_ENDPOINT`
    - `R2_BUCKET`
    - `R2_REGION`
    - `R2_ACCESS_KEY_ID`
    - `R2_SECRET_ACCESS_KEY`
    - `R2_PUBLIC_BASE_URL`

3.  **启动前端服务**:
    ```bash
    # 进入前端目录
    cd packages/web

    # 安装依赖
    pnpm install

    # 运行前端开发服务器
    pnpm run dev
    ```
    前端服务将运行在 `http://localhost:5173`。

### 详细文档

-   关于此系统架构与数据流的详细技术概览，请参阅 **[统一认证系统总结](docs/auth_system_summary.md)**。
-   关于所有 API 端点的快速参考，请参阅 **[API 文档](blueprints/api_documentation.md)**。
