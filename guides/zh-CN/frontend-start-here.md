# 前端开发从这里开始

这份文档给中文前端工程师使用。目标是让你不用先读完整后端代码，也能开始搭建 SkillHub 前端。

详细英文版见：

- [../frontend-start-here.md](../frontend-start-here.md)
- [../frontend-information-architecture.md](../frontend-information-architecture.md)
- [../frontend-integration.md](../frontend-integration.md)
- [../typescript-sdk.md](../typescript-sdk.md)

## 一句话结论

你可以开始做前端页面。后端已经提供了：

- frontend read-model API；
- TypeScript SDK；
- OpenAPI；
- 页面级 contract fixtures；
- auth/session；
- review/promotion/community/release/CI 等主要业务接口。

前端当前缺的是 UI 工程，不是后端能力。

## 推荐技术栈

建议：

- Vue 3
- Vite
- TypeScript
- Vue Router
- Pinia 或 TanStack Query/Vue Query

不要先做营销页。第一个可用页面应该是 skill discovery/search。

## 本地开发准备

前端默认跑在：

```text
http://localhost:5173
```

后端需要允许这个 origin：

```powershell
$env:SKILLHUB_CORS_ALLOWED_ORIGINS = "http://localhost:5173"
```

最小后端启动：

```powershell
cd server
$env:SKILLHUB_DATABASE_URL = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
$env:SKILLHUB_CORS_ALLOWED_ORIGINS = "http://localhost:5173"
$env:SKILLHUB_STORAGE_PROVIDER = "local"
$env:SKILLHUB_STORAGE_ROOT = "./data/storage"
$env:SKILLHUB_LOCAL_MODE = "true"
go run ./cmd/skillhub-server
```

## 引入 SDK

在前端项目的 `package.json`：

```json
{
  "dependencies": {
    "@miqro/skillhub-client": "file:../clients/typescript/skillhub"
  }
}
```

初始化：

```ts
import { SkillHubClient } from "@miqro/skillhub-client";

export const skillhub = new SkillHubClient({
  baseUrl: import.meta.env.VITE_SKILLHUB_API_URL ?? "http://localhost:8080",
  credentials: "include",
});
```

页面加载建议使用：

```ts
const page = await skillhub.unwrap(skillhub.frontendSearch({ keyword: "agent" }));
```

## 第一周开发顺序

按这个顺序做，不要跳：

1. App shell
   - Header
   - Navigation
   - Search box
   - Error boundary
   - SDK provider

2. Auth
   - Login page
   - `client.login(username, password)`
   - `client.me()`
   - 401 跳转 `/login`
   - 403 显示无权限
   - logout 临时 raw fetch `POST /api/v1/auth/logout`

3. Discover/Search
   - `/`
   - `/search`
   - `client.frontendSearch(...)`
   - filters、sort、pagination

4. Skill detail
   - `/skills/:namespace/:slug`
   - `/skills/:namespace/:slug/versions/:version`
   - 使用 frontend read-model，不要自己拼底层接口

5. Namespace
   - `/namespaces`
   - `/namespaces/:namespace`

6. Release + CI
   - release list/detail
   - CI pipeline runs
   - checks / artifacts / gates

7. Community
   - issues
   - discussions
   - wiki
   - proposals

8. Review / Promotion / Governance / Admin
   - review queue/detail
   - promotion queue/detail
   - governance workbench
   - admin dashboard

## 页面开发规则

### 1. 优先用 frontend read-model

例如 skill detail：

```ts
const detail = await skillhub.unwrap(
  skillhub.frontendSkillDetail(namespace, slug),
);
```

不要在页面里自己并发请求很多底层接口再拼数据。

### 2. 权限按钮看 `availableActions`

```ts
if (page.availableActions.canApprove) {
  showApproveButton();
}
```

不要用前端 role string 自己判断。

### 3. mutation 后重新加载 read-model

```ts
await skillhub.unwrap(skillhub.approveReview(reviewId, { comment: "OK" }));
const updated = await skillhub.unwrap(skillhub.frontendReviewDetail(reviewId));
```

### 4. 错误处理

建议页面 loader 用 `unwrap`，表单错误用 envelope mode。

常见状态：

| 状态 | 前端行为 |
|---|---|
| 401 | 跳转登录 |
| 403 | 显示无权限 |
| 404 | Not Found 页面 |
| 409 | 显示冲突原因，例如 gate failed |
| 429 | 显示限流提示 |
| 503 | 显示服务暂不可用，允许重试 |

### 5. 不要无界预取

列表页使用 `page` / `size`。

SDK iterator 只用于有限后台预取，并设置 `maxPages`。

## Mock / Storybook 数据

使用这里的 contract fixtures：

```text
server/internal/http/frontend/testdata/contracts/
```

它们是当前 read-model response envelope 的示例，可以用来做：

- mock API；
- Storybook stories；
- 页面 skeleton；
- 空状态/错误状态设计参考。

## 当前已知缺口

- `web/` 目录还没有正式前端工程。
- TS SDK 还没有发布到 npm，需要本地 file dependency。
- SDK 暂时没有 `logout()` 方法，先 raw fetch。
- CI step logs 暂未持久化。
- 前端页面设计系统还没确定。

这些不是前端开工 blocker。

## 前端 MVP 完成标准

MVP 至少要证明：

- 用户可以登录并看到当前用户状态；
- 可以搜索 skill；
- 可以查看 skill、version、release；
- 可以查看 namespace；
- 可以查看 CI 状态和 gate；
- 可以浏览 community 内容；
- reviewer 可以处理 review/promotion；
- admin/governance 页面能按权限显示；
- 所有按钮权限来自 `availableActions`；
- 所有 mutation 后都会 refetch read-model。

