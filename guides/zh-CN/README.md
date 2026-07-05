# 中文文档入口

这里是中文工程师的快速入口。英文文档仍然是 canonical source，中文文档负责降低上手成本。

## 推荐阅读路线

### 后端/部署工程师

1. [../../README.zh-CN.md](../../README.zh-CN.md)
2. [../backend-quickstart.md](../backend-quickstart.md)
3. [../api-usage.md](../api-usage.md)
4. [../end-to-end-flow.md](../end-to-end-flow.md)

### 前端工程师

1. [frontend-start-here.md](frontend-start-here.md)
2. [../frontend-start-here.md](../frontend-start-here.md)
3. [../frontend-information-architecture.md](../frontend-information-architecture.md)
4. [../frontend-integration.md](../frontend-integration.md)
5. [../typescript-sdk.md](../typescript-sdk.md)

### 只想理解产品

1. [../../README.zh-CN.md](../../README.zh-CN.md)
2. [../end-to-end-flow.md](../end-to-end-flow.md)
3. [../frontend-information-architecture.md](../frontend-information-architecture.md)

## 文档维护原则

- 英文文档维护完整接口、参数、类型和契约。
- 中文文档维护上手路径、开发顺序、注意事项。
- 不要把所有英文文档完整复制成中文，避免长期不同步。
- 如果中文文档与英文文档冲突，以英文文档和代码为准。

## 当前最重要的事实

- 后端功能已经足够前端开工。
- 前端不是缺后端接口，而是还没有 UI 工程。
- 前端应该从 read-model API 和 TypeScript SDK 开始。
- 权限不要在前端推导，使用 `availableActions`。
- 生产环境 Redis/S3 是真实接线能力，不是占位说明。

