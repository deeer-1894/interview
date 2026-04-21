# Mock Interview Workspace

一个基于 `CloudWeGo Eino ADK` 的模拟技术面试工作台。项目当前已经不是单次 `/chat` 问答 Demo，而是围绕 `Conversation -> Task -> Run` 组织的可恢复面试系统，包含流式执行、澄清中断、评分复盘、候选人画像、材料管理和技能驱动。

## 当前能力

- 会话式工作区
  - 创建、置顶、归档、重命名、删除工作区
  - 每个工作区聚合 `Task` 与 `Run`
- 面试运行时
  - 创建 run、流式输出、取消、恢复
  - `waiting_clarify` 中断与 resume
  - checkpoint 恢复优先，transcript 续跑兜底
- 面试策略
  - `standard`
  - `stress`
  - `weakness_focused`
  - `system_design`
  - `resume_deep_dive`
- 复盘中心
  - scorecard
  - 决策审计
  - 追问树
  - phase view
  - 候选人画像
- 技能与材料
  - 本地 `skills/` 技能库
  - 技能创建、更新、上传
  - 文本材料创建、上传、下载、更新、删除
  - 材料默认按 task / run 绑定
- 辅助能力
  - 显式开关控制的联网检索
  - `copilot` 提示接口
  - SSE 事件流与 review snapshot 聚合查询

## 架构总览

```text
web/src (React + Vite workbench)
        │
        ▼
internal/web
HTTP API + SSE + static hosting
        │
        ▼
internal/control/service
conversation / task / run lifecycle
        │
        ▼
internal/control/runtime
engine + run context + telemetry
        │
        ▼
internal/control/middleware
output -> setup -> checkpoint -> clarify -> skill -> memory
-> reduction -> artifact_binding -> planning -> adversarial
-> tool_routing -> summarization
        │
        ├─ internal/executors/interview
        ├─ internal/interview
        ├─ internal/tools
        └─ internal/state + internal/storage/artifacts
```

## 目录

```text
cmd/mockinterview/                CLI 与 HTTP server 入口
internal/control/                生命周期、运行时、中间件、workflow
internal/executors/interview/    面试执行器、评分生成、流式输出
internal/interview/              面试领域模型、prompt、信号、决策、skill pack
internal/interview/adkapp/       Eino ADK 接线与模型工厂
internal/protocol/               API / 事件 / review 共享协议
internal/state/                  Mongo / Redis 适配与仓储
internal/storage/artifacts/      附件文件存储
internal/tools/                  gateway、registry、MCP HTTP transport
internal/web/                    REST、SSE、前端静态托管
scripts/                         smoke / UI smoke 脚本
skills/                          项目内技能库
web/                             React + Vite 前端
```

## 依赖

- Go：使用 [go.mod](/Users/shiyi/mockinterview/go.mod) 中声明的版本
- Node.js：用于前端开发与测试
- Redis：checkpoint、clarify、短期运行态
- MongoDB：conversation、task、run、message、event、profile、artifact metadata

## 环境变量

### 持久化层

- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_PREFIX`
- `MONGO_URI`
- `MONGO_DATABASE`
- `ARTIFACT_STORAGE_DIR`

### 模型配置

- `MODEL_PROVIDER`
- `MODEL_NAME`
- `MODEL_API_KEY`
- `MODEL_BASE_URL`
- `MODEL_TIMEOUT_SECONDS`

同时兼容部分 provider 专属变量，例如：

- `OPENAI_API_KEY`
- `CLAUDE_API_KEY` / `ANTHROPIC_API_KEY`
- `GEMINI_API_KEY`
- `DEEPSEEK_API_KEY`
- `OLLAMA_BASE_URL`
- `QWEN_API_KEY`
- `OPENAI_BY_AZURE`
- `CLAUDE_BY_BEDROCK`
- `CLAUDE_BY_VERTEX`

支持的 provider：

- `openai`
- `openai-compatible`
- `claude`
- `gemini`
- `deepseek`
- `ollama`
- `qwen`

### 调试

- `STREAM_DEBUG=true`

## 启动

### 1. 启动后端

先准备 Redis、Mongo 和模型环境变量，然后运行：

```bash
REDIS_ADDR=localhost:6379 \
MONGO_URI=mongodb://localhost:27017 \
MONGO_DATABASE=mockinterview \
MODEL_PROVIDER=openai-compatible \
MODEL_NAME=your-model \
MODEL_API_KEY=your-api-key \
MODEL_BASE_URL=https://your-endpoint \
go run ./cmd/mockinterview -serve -addr :8080
```

项目里也提供了 [start.sh](/Users/shiyi/mockinterview/start.sh) 和 [start-glm.sh](/Users/shiyi/mockinterview/start-glm.sh)。`start.sh` 在当前分支只从环境变量读取模型配置；`start-glm.sh` 提供了一个带 GLM 默认 provider/model/base URL 的便捷入口，但同样要求通过环境变量提供 API key。

### 2. 启动前端开发模式

```bash
cd /Users/shiyi/mockinterview/web
npm install
npm run dev
```

### 3. 构建前端

```bash
cd /Users/shiyi/mockinterview/web
npm run build
```

构建后的 `web/dist` 会由 Go 服务直接托管。

### 4. CLI 单轮运行

不加 `-serve` 时可以直接从命令行跑一轮面试：

```bash
go run ./cmd/mockinterview \
  -skill go-agent-interview-sim \
  -mode standard \
  -prompt "请模拟一场 Go agent 开发岗位的技术面试。"
```

## 测试

### 后端

```bash
go test ./...
```

### 接口级 smoke

后端启动后：

```bash
bash /Users/shiyi/mockinterview/scripts/smoke_test.sh
```

脚本覆盖：

- `health / profile / skills`
- conversation 生命周期
- files 创建、上传、读取、更新、删除
- task / run / review / events
- cancel / resume
- 兼容接口 `POST /api/interview`

### 前端

```bash
cd /Users/shiyi/mockinterview/web
npm run test:components
npm run test:e2e
```

### 页面级 UI smoke

```bash
bash /Users/shiyi/mockinterview/scripts/ui_smoke_test.sh
```

更多手工验收项见 [MANUAL_ACCEPTANCE_CHECKLIST.md](/Users/shiyi/mockinterview/MANUAL_ACCEPTANCE_CHECKLIST.md)。

## 核心 API

### 会话与任务

- `GET /api/conversations`
- `POST /api/conversations`
- `GET /api/conversations/:id`
- `PATCH /api/conversations/:id`
- `DELETE /api/conversations/:id`
- `POST /api/tasks`

### 运行

- `POST /api/runs`
- `GET /api/runs/:id`
- `POST /api/runs/:id/resume`
- `POST /api/runs/:id/cancel`
- `POST /api/runs/:id/copilot`
- `GET /api/runs/:id/review`
- `GET /api/runs/:id/events`

### 技能与材料

- `GET /api/skills`
- `POST /api/skills`
- `GET /api/skills/:name`
- `PUT /api/skills/:name`
- `GET /api/files?conversationId=...`
- `POST /api/files`
- `GET /api/files/:id`
- `GET /api/files/:id?content=1`
- `GET /api/files/:id?download=1`
- `PUT /api/files/:id`
- `DELETE /api/files/:id`

### 其他

- `GET /api/health`
- `GET /api/profile`
- `POST /api/interview`

`POST /api/interview` 目前仍保留兼容路径，但主流程应优先使用 `Conversation / Task / Run` API。

## 前端工作台

`web/src/app.tsx` 当前承载以下主路径：

- 左侧工作区列表与状态排序
- 中央消息画布与 Markdown 流式渲染
- 启动 run 弹层
- 会话配置弹层
- 会话内联评分 / 阶段 / 画像摘要
- copilot hint 面板

会话内联结果主入口位于：

- [web/src/app.tsx](/Users/shiyi/mockinterview/web/src/app.tsx)
- [web/src/components/session/session-context-modal.tsx](/Users/shiyi/mockinterview/web/src/components/session/session-context-modal.tsx)

## 相关文档

- [ARCHITECTURE_EXECUTION_STANDARD.md](/Users/shiyi/mockinterview/ARCHITECTURE_EXECUTION_STANDARD.md)：当前架构边界与执行标准
- [MANUAL_ACCEPTANCE_CHECKLIST.md](/Users/shiyi/mockinterview/MANUAL_ACCEPTANCE_CHECKLIST.md)：手工验收清单
- [MANUAL_ACCEPTANCE_RESULT.md](/Users/shiyi/mockinterview/MANUAL_ACCEPTANCE_RESULT.md)：最近一次手工验收结果
