# Cyime 初始化工具

## 使用方法

### 首次初始化（配置 SSO 登录）

```bash
cd packages/server
go run cmd/init/main.go
```

或者使用编译好的二进制文件：

```bash
./init
```

### 工具功能

1. **数据库初始化** - 自动创建所有必需的数据库表
2. **SSO 配置向导** - 交互式配置 OAuth/SSO 登录提供商
   - GitHub OAuth
   - Google OAuth
   - 自定义 OIDC 提供商

### 配置 GitHub OAuth

1. 访问 https://github.com/settings/developers
2. 点击 "New OAuth App"
3. 填写应用信息：
   - **Application name**: Cyime
   - **Homepage URL**: http://localhost:8080
   - **Authorization callback URL**: http://localhost:8080/api/v1/auth/callback/github
4. 创建后获取 Client ID 和 Client Secret
5. 运行初始化工具，选择 "1. GitHub"
6. 输入 Client ID 和 Client Secret

### 配置 Google OAuth

1. 访问 https://console.cloud.google.com/apis/credentials
2. 点击 "Create Credentials" → "OAuth client ID"
3. 应用类型选择 "Web application"
4. 添加授权重定向 URI：
   - http://localhost:8080/api/v1/auth/callback/google
5. 创建后获取 Client ID 和 Client Secret
6. 运行初始化工具，选择 "2. Google"
7. 输入 Client ID 和 Client Secret

### 配置自定义 OIDC 提供商

1. 运行初始化工具
2. 选择 "3. 自定义 OIDC 提供商"
3. 按提示输入：
   - 提供商名称
   - Issuer URL
   - Auth URL
   - Token URL
   - UserInfo URL
   - Client ID
   - Client Secret
   - Scopes

### 跳过配置

如果暂时不想配置 SSO，可以选择 "4. 跳过"，稍后可以通过以下方式手动添加：

```bash
# 使用 SQLite 命令行工具
sqlite3 ~/.cyimewrite/cyimewrite.db

# 插入 OAuth 提供商配置
INSERT INTO auth_providers (id, name, protocol_type, issuer_url, auth_url, token_url, user_info_url, client_id, client_secret_encrypted, icon_url, scopes, is_active, created_at, updated_at)
VALUES (
  'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx',
  'github',
  'oauth2',
  NULL,
  'https://github.com/login/oauth/authorize',
  'https://github.com/login/oauth/access_token',
  'https://api.github.com/user',
  'your-client-id',
  'your-client-secret',
  'https://github.com/fluidicon.png',
  'read:user user:email',
  1,
  datetime('now'),
  datetime('now')
);
```

## 下一步

初始化完成后，启动服务器：

```bash
go run cmd/server/main.go
```

访问 http://localhost:8080 测试登录功能。

## 注意事项

⚠️ **安全提示**：
- 当前版本 Client Secret 以明文存储在数据库中
- 生产环境请实现加密存储（使用环境变量或加密算法）
- 建议使用 HTTPS 保护回调 URL

## 故障排除

### 数据库已存在

如果数据库已存在但想重新初始化：

```bash
# 删除现有数据库
rm ~/.cyimewrite/cyimewrite.db

# 重新运行初始化工具
go run cmd/init/main.go
```

### 提供商已存在错误

如果提示提供商已存在，可以先删除再重新添加：

```bash
sqlite3 ~/.cyimewrite/cyimewrite.db "DELETE FROM auth_providers WHERE name = 'github';"
```

然后重新运行初始化工具。
