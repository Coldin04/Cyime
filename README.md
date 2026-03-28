# CyimeWrite

🍋 CyimeWrite —— 青柠写  轻量简洁的云端同步Markdown编辑器 为随心流动的创意而生

[![Ask DeepWiki](https://devin.ai/assets/askdeepwiki.png)](https://deepwiki.com/Coldin04/CyimeWrite)
[![zread](https://img.shields.io/badge/Ask_Zread-_.svg?style=for-the-badge&color=00b0aa&labelColor=000000&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTQuOTYxNTYgMS42MDAxSDIuMjQxNTZDMS44ODgxIDEuNjAwMSAxLjYwMTU2IDEuODg2NjQgMS42MDE1NiAyLjI0MDFWNC45NjAxQzEuNjAxNTYgNS4zMTM1NiAxLjg4ODEgNS42MDAxIDIuMjQxNTYgNS42MDAxSDQuOTYxNTZDNS4zMTUwMiA1LjYwMDEgNS42MDE1NiA1LjMxMzU2IDUuNjAxNTYgNC45NjAxVjIuMjQwMUM1LjYwMTU2IDEuODg2NjQgNS4zMTUwMiAxLjYwMDEgNC45NjE1NiAxLjYwMDFaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00Ljk2MTU2IDEwLjM5OTlIMi4yNDE1NkMxLjg4ODEgMTAuMzk5OSAxLjYwMTU2IDEwLjY4NjQgMS42MDE1NiAxMS4wMzk5VjEzLjc1OTlDMS42MDE1NiAxNC4xMTM0IDEuODg4MSAxNC4zOTk5IDIuMjQxNTYgMTQuMzk5OUg0Ljk2MTU2QzUuMzE1MDIgMTQuMzk5OSA1LjYwMTU2IDE0LjExMzQgNS42MDE1NiAxMy43NTk5VjExLjAzOTlDNS42MDE1NiAxMC42ODY0IDUuMzE1MDIgMTAuMzk5OSA0Ljk2MTU2IDEwLjM5OTlaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik0xMy43NTg0IDEuNjAwMUgxMS4wMzg0QzEwLjY4NSAxLjYwMDEgMTAuMzk4NCAxLjg4NjY0IDEwLjM5ODQgMi4yNDAxVjQuOTYwMUMxMC4zOTg0IDUuMzEzNTYgMTAuNjg1IDUuNjAwMSAxMS4wMzg0IDUuNjAwMUgxMy43NTg0QzE0LjExMTkgNS42MDAxIDE0LjM5ODQgNS4zMTM1NiAxNC4zOTg0IDQuOTYwMVYyLjI0MDFDMTQuMzk4NCAxLjg4NjY0IDE0LjExMTkgMS42MDAxIDEzLjc1ODQgMS42MDAxWiIgZmlsbD0iI2ZmZiIvPgo8cGF0aCBkPSJNNCAxMkwxMiA0TDQgMTJaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00IDEyTDEyIDQiIHN0cm9rZT0iI2ZmZiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8L3N2Zz4K&logoColor=ffffff)](https://zread.ai/Coldin04/CyimeWrite)


## 仓库说明

关于仓库迁移、提交历史整理以及部署仓库说明，请参阅：

- [仓库迁移与提交历史说明](docs/repository_migration_note.md)

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

    # 建议先从仓库根目录的 .env.example 复制一份
    # cp ../.env.example .env

    # 运行后端
    go run ./cmd/server/main.go
    ```
    后端服务将运行在 `http://localhost:8080`。
    说明：后端启动会自动读取 `packages/server/.env`。

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

### 关键环境变量

完整示例见根目录 [`.env.example`](/home/coldin04/CyimeWrite/.env.example)。下面只列当前最常用的配置项。

- 认证与会话
  - `JWT_SECRET_KEY`：JWT 与部分签名逻辑依赖的密钥，生产环境必须配置。
  - `ACCESS_TOKEN_LIFETIME_MINUTES`：Access Token 生命周期，默认 `15`。
  - `REFRESH_TOKEN_LIFETIME_HOURS`：Refresh Token 生命周期，默认 `720`（30 天）。
  - `FRONTEND_CALLBACK_URL`：登录成功后回跳到前端的地址。

- 文档数量限制
  - `DEFAULT_DOCUMENT_QUOTA`：全局默认文档上限。
  - 留空表示不限制。
  - 用户如果单独配置了 `document_quota`，会优先使用用户自己的值。
  - 后端会在创建文档时校验这个上限。

- 媒体与图床
  - `MEDIA_STORAGE_PROVIDER`：可选 `local | r2 | s3 | cos`，默认 `local`。
  - `MEDIA_TOKEN_SECRET`：私有媒体短期签名密钥；留空时回退到 `JWT_SECRET_KEY`。
  - `MEDIA_SIGN_TTL_SECONDS`：私有媒体签名有效期，默认 `120` 秒。
  - `MEDIA_AVATAR_SIGN_TTL_SECONDS`：私有头像签名有效期，默认 `300` 秒。
  - `MEDIA_AVATAR_MAX_BYTES`：头像上传大小限制，默认 `2MB`。

- 本地媒体存储
  - `MEDIA_LOCAL_ROOT_DIR`：本地文件落盘目录。
  - `MEDIA_LOCAL_BASE_URL`：本地媒体对外访问前缀。

- S3 / R2 / COS 兼容存储
  - `MEDIA_S3_ENDPOINT`
  - `MEDIA_S3_BUCKET`
  - `MEDIA_S3_REGION`
  - `MEDIA_S3_ACCESS_KEY_ID`
  - `MEDIA_S3_SECRET_ACCESS_KEY`
  - `MEDIA_S3_PUBLIC_BASE_URL`
  - 兼容旧变量：`R2_ENDPOINT`、`R2_BUCKET`、`R2_REGION`、`R2_ACCESS_KEY_ID`、`R2_SECRET_ACCESS_KEY`、`R2_PUBLIC_BASE_URL`

- 媒体 GC / 引用回收
  - `MEDIA_ASSET_GC_ENABLED`
  - `MEDIA_ASSET_GC_INTERVAL`
  - `MEDIA_ASSET_GC_BATCH_SIZE`
  - `MEDIA_ASSET_GC_MAX_ATTEMPTS`
  - `MEDIA_ASSET_GC_RETRY_GAP`
  - `MEDIA_ASSET_RECONCILE_ENABLED`
  - `MEDIA_ASSET_RECONCILE_BATCH_SIZE`

- 其他
  - `CORS_ALLOWED_ORIGINS`：允许的前端来源，多个值用英文逗号分隔。
  - `RESET_WORKSPACE_TABLES_ON_BOOT`：启动时重置业务表，仅调试时使用。

### 详细文档

-   关于此系统架构与数据流的详细技术概览，请参阅 **[统一认证系统总结](docs/auth_system_summary.md)**。
-   关于所有 API 端点的快速参考，请参阅 **[API 文档](blueprints/api_documentation.md)**。
