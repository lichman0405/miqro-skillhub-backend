# Miqro-SkillHub

自托管 Agent Skill Registry 后端，用于发布、审核、发布 release、搜索、下载和安装 agent skills。

> 英文文档是接口与实现细节的 canonical source。中文文档用于快速上手和团队协作入口，细节以英文文档为准。

## 这个项目是什么

Miqro-SkillHub 是一个面向 agent skills 的 hub，不是 GitHub clone。

它关注的是：

- skill 包上传与校验；
- namespace / team 组织；
- skill 版本、release、download；
- review / promotion / governance；
- CI/CD gate 检查；
- community 功能：issues、discussions、wiki、proposals；
- frontend read-model API；
- TypeScript SDK；
- tool / CLI / agent 集成接口。

## 当前状态

后端已经具备前端开发所需的主要能力：

- Go SDK-first 后端；
- PostgreSQL schema / migrations / repositories；
- auth、RBAC、namespace、skill lifecycle；
- package validation、release、review、promotion；
- agent CI/CD；
- community 功能；
- frontend read-model endpoints；
- OpenAPI；
- TypeScript SDK；
- Redis-backed sessions 和 distributed rate limiting；
- S3/MinIO object storage；
- Docker Compose 本地与 release 示例。

`web/` 目录仍为空。这个仓库目前提供后端、SDK、接口文档和前端对接指南，不包含正式前端 UI。

## 中文工程师先读什么

推荐阅读顺序：

1. [guides/zh-CN/README.md](guides/zh-CN/README.md) — 中文文档入口。
2. [guides/zh-CN/frontend-start-here.md](guides/zh-CN/frontend-start-here.md) — 前端工程师第一周如何开工。
3. [guides/frontend-start-here.md](guides/frontend-start-here.md) — 英文前端入口，细节更完整。
4. [guides/frontend-information-architecture.md](guides/frontend-information-architecture.md) — 页面、路由、app shell。
5. [guides/frontend-integration.md](guides/frontend-integration.md) — 每个页面对应的 SDK/API。
6. [guides/typescript-sdk.md](guides/typescript-sdk.md) — TypeScript SDK 用法。
7. [guides/api-usage.md](guides/api-usage.md) — 原始 HTTP API 参考。

## 本地快速启动后端

最小本地模式：

```powershell
cd server
$env:SKILLHUB_DATABASE_URL = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
$env:SKILLHUB_CORS_ALLOWED_ORIGINS = "http://localhost:5173"
$env:SKILLHUB_STORAGE_PROVIDER = "local"
$env:SKILLHUB_STORAGE_ROOT = "./data/storage"
$env:SKILLHUB_LOCAL_MODE = "true"
go run ./cmd/skillhub-server
```

完整本地基础设施：

```powershell
docker compose up -d postgres redis minio
docker compose --profile full up -d
```

Windows 上如果没有 Docker，也可以使用已有 PostgreSQL 或让别人提供远程后端。

## 前端怎么接

前端默认建议：

- Vue 3
- Vite
- TypeScript
- Vue Router
- Pinia 或 TanStack Query/Vue Query
- 本地引用 TypeScript SDK

SDK 依赖：

```json
{
  "dependencies": {
    "@miqro/skillhub-client": "file:../clients/typescript/skillhub"
  }
}
```

SDK 初始化：

```ts
import { SkillHubClient } from "@miqro/skillhub-client";

export const skillhub = new SkillHubClient({
  baseUrl: "http://localhost:8080",
  credentials: "include",
});
```

前端页面优先使用 frontend read-model API，不要自己拼很多底层接口。

## 前端开发关键规则

1. 用 `SkillHubClient`，不要一开始就手写大量 `fetch`。
2. 页面加载用 `client.unwrap(...)`。
3. 表单错误展示可以用 envelope mode。
4. 权限按钮只看后端返回的 `availableActions`。
5. mutation 成功后必须重新拉取页面 read model。
6. 不要在前端自己推导 RBAC。
7. 不要无界预取 review/promotion/community 队列。
8. logout 目前 SDK 还没有公开方法，可以临时 raw fetch `POST /api/v1/auth/logout`。

## 生产部署注意

生产模式必须：

- 设置 `SKILLHUB_LOCAL_MODE=false`；
- 使用 PostgreSQL；
- 使用 S3/MinIO object storage；
- 使用 Redis-backed sessions；
- 使用 Redis-backed rate limiting；
- 设置明确的 `SKILLHUB_CORS_ALLOWED_ORIGINS`；
- 不使用 `minioadmin:minioadmin`、`skillhub:skillhub` 这类本地默认凭据。

Release compose 示例见：

- [compose.release.yml](compose.release.yml)
- [guides/backend-quickstart.md](guides/backend-quickstart.md)

## 前端开发者网页文档

`site/` 目录下的静态站点提供了前端对接文档。包括 frontend read-model API、TypeScript SDK、本地开发和 contract fixtures 说明。

- **本地查看：** 直接打开 `site/index.html`。
- **GitHub Pages：** push 到 `master` 后，`.github/workflows/pages.yml` 会自动部署。部署成功后可通过仓库的 Pages URL（如 `https://lichman0405.github.io/miqro-skillhub-backend/`）访问。
- **启用 Pages：** 如果 workflow 成功后站点未显示，需要在 GitHub 仓库 **Settings → Pages** 中将来源设为 **GitHub Actions**。

## 当前已知非目标

- 没有正式前端 UI；
- TypeScript SDK 尚未发布到 npm；
- 没有 Kubernetes manifests；
- 没有生产级 secrets 管理方案；
- CI step logs 暂未持久化；
- Dockerfile 当前仍需进一步做 non-root runtime hardening。

## 推送前验证

常用验证：

```powershell
cd server
go test ./...
go vet ./...
go build ./cmd/skillhub-server
go build ./cmd/skillhub-migrate
go build ./cmd/skillhub-worker
```

TypeScript SDK：

```powershell
cd clients/typescript/skillhub
npm run build
npm test
```

