# 图床 Provider 贡献指南

## 目标

把内置图床做成声明式注册，贡献者新增一个 provider 时优先只改一个 JSON 文件。

当前实现已支持：

- 前端按 provider `fields` 动态渲染配置表单
- 用户配置按 `fieldValues` 通用存储
- 后端上传器按 provider `upload` 规则执行请求和解析响应

## 目录

- provider 注册与加载：`packages/server/internal/imagebeds/registry.go`
- provider 声明文件：`packages/server/internal/imagebeds/providers/*.json`
- 通用上传执行器：`packages/server/internal/media/document_image_upload.go`
- 用户配置页面：`packages/web/src/lib/components/user/ImageBedsTab.svelte`

## 新增 Provider（默认流程）

1. 在 `packages/server/internal/imagebeds/providers/` 新增一个 JSON 文件。
2. 定义 `providerType`、`fields`、`runtime`、`upload`。
3. 重启服务后，用户中心会自动出现新 provider，配置后可直接用于文档粘贴上传。

如果你的目标图床满足现有声明能力（multipart + headers/query/formFields + JSON 路径解析），不需要再改 Go 代码。

## JSON 结构示例（ImgBB）

```json
{
  "providerType": "imgbb",
  "displayName": "ImgBB",
  "description": "ImgBB image hosting",
  "fields": [
    {
      "key": "apiToken",
      "type": "password",
      "label": "API Key",
      "placeholder": "Enter ImgBB API key",
      "required": true
    }
  ],
  "runtime": {
    "defaultBaseUrl": "https://api.imgbb.com/1",
    "baseUrlEnv": "IMGBB_API_BASE_URL",
    "apiTokenEnv": "IMGBB_API_KEY"
  },
  "upload": {
    "method": "POST",
    "urlTemplate": "{{baseUrl}}/upload",
    "fileField": "image",
    "headers": [
      { "key": "Accept", "valueTemplate": "application/json" }
    ],
    "query": [
      { "key": "key", "valueTemplate": "{{apiToken}}", "required": true }
    ],
    "formFields": [],
    "successJsonPath": "success",
    "successEquals": "true",
    "resultUrlPaths": ["data.url"],
    "errorMessagePaths": ["error.message", "message"]
  }
}
```

## 字段规则

- `fields[*].key` 会作为模板变量名，供 `{{key}}` 使用。
- 用户填写的字段会写入 `fieldValues`，并用于 `query` / `headers` / `formFields` 渲染。
- `fields[*].type` 支持 `text` / `password` / `url` / `number`（用于基础校验）。

## 设计边界

- 公共图床是 upload-only：只返回 URL，不做删除和生命周期管理。
- 私有受控媒体（`managed-r2`）继续走现有资产体系。
- 若某图床协议超出现有声明能力，再单独补专用逻辑；不要先把主流程改回大量 `if/switch`。
