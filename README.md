# OfferBot Mock Interview

OfferBot 是一个基于 Go、CloudWeGo Eino ADK 和 React 的 AI 模拟面试系统。当前版本聚焦一件事：围绕候选人简历和目标职位，生成更接近真实技术面的长流程追问，并通过 SSE 把模型输出实时推送到前端。

## 当前状态

- 后端：Go + Gin + Eino ADK
- 前端：React + Vite + Tailwind + shadcn/ui 风格组件
- 存储：MongoDB 持久化简历、会话、消息、评分；Redis 保存 checkpoint
- 模型：通过 `MODEL_PROVIDER` / `MODEL_NAME` / `MODEL_API_KEY` / `MODEL_BASE_URL` 接入
- 认证：当前开发账号固定为 `shiyi123@123.com` / `shiyi123456`

## 核心能力

- 简历驱动面试
  - 保存候选人真实简历
  - 创建不同职位、级别、模式的面试会话
  - 面试问题基于当前会话上下文和简历内容生成

- Eino ADK Runtime
  - 使用 `ChatModelAgent` 承载模型调用
  - 使用 middleware 做上下文注入、上下文隔离、工具安全、追问质量控制
  - 支持 checkpoint、clarify interrupt、resume 这类长任务控制点

- 流式对话
  - `POST /api/sessions/stream` 创建会话并流式生成开场问题
  - `POST /api/sessions/:id/messages/stream` 提交回答并流式生成下一轮追问
  - 前端展示“正在思考 / 正在查看上下文 / 正在生成”状态
  - 后端避免空 assistant 静默成功，前端也会对空输出给出可见提示

- ChatGPT 风格前端
  - 左侧会话列表、搜索弹层、删除会话
  - 中央 Markdown 消息流
  - 底部固定输入框
  - 深浅色切换

## 架构

```text
web/
  React + Vite UI
  Markdown 渲染 / SSE 流式消费 / 会话搜索与管理

cmd/mockinterview/
  Gin HTTP API
  登录、简历、会话、消息、评分接口

internal/interview/runtime/
  Eino ADK Runtime
  model factory
  middleware
  graph tools
  clarify tool

internal/interview/session/
  面试会话与消息模型

internal/interview/resume/
  简历模型

internal/interview/report/
  评分结果模型

internal/state/mongo/
  MongoDB store

internal/state/redis/
  Redis checkpoint store
```

## 目录说明

```text
cmd/mockinterview/                 后端入口与 HTTP API
docs/eino-runtime-notes.md         Eino / Eino ADK 设计笔记
internal/interview/runtime/        Agent Runtime、middleware、工具与模型工厂
internal/interview/resume/         简历领域模型
internal/interview/session/        面试会话、消息与状态模型
internal/interview/report/         面试评分模型
internal/interview/store/          存储接口
internal/state/memory/             内存存储实现
internal/state/mongo/              MongoDB 存储实现
internal/state/redis/              Redis checkpoint 实现
web/                               React 前端
```

## 环境变量

### 存储

- `MONGO_URI`，默认 `mongodb://localhost:27017`
- `MONGO_DATABASE`，默认 `mockinterview`
- `REDIS_ADDR`，默认 `localhost:6379`
- `REDIS_PASSWORD`

### 模型

- `MODEL_PROVIDER`
- `MODEL_NAME`
- `MODEL_API_KEY`
- `MODEL_BASE_URL`
- `MODEL_TIMEOUT_SECONDS`

当前后端会从环境变量创建 Eino ChatModel。推荐使用 openai-compatible provider 接入兼容 OpenAI Chat Completions 的模型服务。

## 启动

### 后端

```bash
MODEL_PROVIDER=openai-compatible \
MODEL_NAME=your-model \
MODEL_API_KEY=your-api-key \
MODEL_BASE_URL=https://your-model-endpoint \
MONGO_URI=mongodb://localhost:27017 \
MONGO_DATABASE=mockinterview \
REDIS_ADDR=localhost:6379 \
go run ./cmd/mockinterview -serve -addr :8080
```

本地也可以使用项目内启动脚本，但脚本里的本机配置不作为通用部署约定。

### 前端

```bash
cd web
npm install
npm run dev
```

默认访问：

- 前端：`http://localhost:5173`
- 后端：`http://localhost:8080`

## API

### 认证

- `POST /api/login`

### 简历

- `GET /api/profile`
- `POST /api/profile`

### 技能

- `GET /api/skills`

### 会话

- `GET /api/sessions`
- `POST /api/sessions`
- `POST /api/sessions/stream`
- `DELETE /api/sessions/:id`

### 消息

- `GET /api/sessions/:id/messages`
- `POST /api/sessions/:id/messages`
- `POST /api/sessions/:id/messages/stream`
- `GET /api/sessions/:id/stream`

### Resume 与评分

- `POST /api/sessions/:id/resume`
- `GET /api/sessions/:id/report`

## 开发验证

### 后端

```bash
go test ./...
```

### 前端

```bash
cd web
npm run build
```

## 设计原则

- 不把具体简历内容写死进提示词或策略代码。
- Skill / middleware 只描述通用面试能力、质量控制和上下文边界。
- 真实问题生成交给模型，工程侧负责上下文、约束、状态和输出质量。
- 空模型输出必须显式失败或可见提示，不能静默成功。
- 简历、职位、会话历史必须按 session 隔离，避免不同职位或项目之间互相污染。
