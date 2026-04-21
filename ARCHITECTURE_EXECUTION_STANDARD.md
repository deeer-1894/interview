# MockInterview Architecture Execution Standard

最后校对时间：`2026-04-20`

这份文档描述的是当前仓库已经落地并应继续遵守的执行标准，不再把项目写成“待迁移蓝图”。如果代码结构发生变化，应先更新本文件，再推进实现。

## 1. 目标

`mockinterview` 的主业务模型是：

- `Conversation`
- `Task`
- `Run`

系统目标是提供一个可持续追问、可中断恢复、可复盘、可沉淀画像的面试工作台，而不是单次文本问答接口。

## 2. 当前分层

```text
web/src
  React + Vite workbench

internal/web
  REST API / SSE / static hosting

internal/control/service
  conversation / task / run lifecycle
  persistence orchestration
  review snapshot aggregation
  event broker

internal/control/runtime
  engine
  run context
  middleware chain
  telemetry spans

internal/control/middleware
  setup / checkpoint / clarify / skill / memory / reduction
  artifact binding / planning / adversarial / tool routing / summarization

internal/executors/interview
  streaming execution
  resume handling
  scorecard generation

internal/interview
  domain model
  prompt strategy
  signal analysis
  decision logic
  replay / insights / skill pack

internal/tools
  gateway
  registry
  MCP HTTP transport

internal/state
  Mongo repositories
  Redis repositories

internal/storage/artifacts
  file payload storage
```

## 3. 边界要求

### 3.1 `internal/web`

职责：

- 请求解析
- 响应序列化
- 路由分发
- SSE 输出
- 前端静态资源托管

约束：

- handler 不直接拼接面试 prompt
- handler 不直接调用底层 ADK runner
- handler 通过 `service.App` 进入业务流程

### 3.2 `internal/control/service`

职责：

- `Conversation / Task / Run` 生命周期
- 事件广播与订阅
- run 恢复、取消、失败收尾
- review snapshot 聚合
- 对 repository、gateway、engine 的编排

约束：

- service 可以协调多个仓储，但不承载 prompt 细节
- service 不直接写前端视图模型

### 3.3 `internal/control/runtime`

职责：

- 运行引擎
- middleware 依赖校验
- 执行上下文
- span / 事件级可观测性

约束：

- runtime 通过 `RunContext` 传递状态
- runtime 不感知 HTTP

### 3.4 `internal/control/middleware`

职责：

- 把运行前准备、决策、工具接入和执行后归档拆成明确阶段

约束：

- middleware 必须声明稳定名称
- 如存在依赖，必须通过 `Spec().Requires` 显式表达
- middleware 只做单一阶段职责，不把整条运行链塞进一个处理器

### 3.5 `internal/executors/interview`

职责：

- 驱动面试执行
- 处理流式 assistant 输出
- 生成评分与学习计划
- 支持 resume 与 transcript-aware fallback

约束：

- executor 不直接处理 HTTP、SSE 或数据库细节

### 3.6 `internal/interview`

职责：

- interview domain 类型
- phase / mode / persona / signal / decision
- prompt 组装策略
- skill pack 与画像反馈

约束：

- domain 不依赖 Web 层
- domain 不依赖具体持久化实现

### 3.7 `internal/tools`

职责：

- tool gateway 抽象
- registry 与 provider 选择
- 远程 provider 链接与 fallback

约束：

- 运行时需要的 tool 能力统一走 gateway
- 不能把存储读写重新塞回 executor 或 handler

### 3.8 `internal/state` 与 `internal/storage/artifacts`

职责：

- Redis 保存 checkpoint、clarify、短期运行态
- Mongo 保存 conversation、task、run、message、event、profile、artifact metadata
- 文件内容通过 artifact file store 落盘

约束：

- 可恢复运行态不能只存在内存
- 大文件内容不直接塞进 Mongo 文档

## 4. 当前运行链路

默认 middleware 顺序以 [internal/control/middleware/middleware.go](/Users/shiyi/mockinterview/internal/control/middleware/middleware.go) 为准：

1. `output`
2. `setup`
3. `checkpoint`
4. `clarify`
5. `skill`
6. `memory`
7. `reduction`
8. `artifact_binding`
9. `planning`
10. `adversarial`
11. `tool_routing`
12. `summarization`
13. `executor`

说明：

- `output` 负责 run.started、人格事件、最终输出持久化和失败/取消收尾
- `setup` 负责标准化 task / run 输入
- `checkpoint` 负责 resume 载入与关键节点保存
- `clarify` 负责缺失输入时挂起为 `waiting_clarify`
- `skill` 负责解析 skill / rubric
- `memory` 与 `reduction` 负责历史摘要压缩
- `artifact_binding` 负责 task / run 级材料绑定
- `planning` 负责生成执行计划
- `adversarial` 负责 phase-aware 的策略扰动
- `tool_routing` 负责显式联网研究等工具接入
- `summarization` 负责运行后摘要回写
- `executor` 由 runtime 在链尾统一调用

## 5. 运行时状态标准

### 5.1 Run Status

允许的主状态：

- `created`
- `running`
- `waiting_clarify`
- `resuming`
- `completed`
- `failed`
- `cancelled`

### 5.2 Run Phase

运行编排阶段：

- `initial`
- `interviewing`
- `evaluating`
- `study_plan`

### 5.3 Interview Phase

面试内部阶段：

- `warmup`
- `probe`
- `adversarial`
- `stress`
- `wrapup`

要求：

- `RunPhase` 与 `InterviewPhase` 必须继续分离
- UI、review 和 replay 可以同时消费两层状态，但不得混为一个字段

## 6. 事件标准

事件是运行时默认遥测协议。新增重要流程时，优先补事件，再补 UI 消费。

当前核心事件类型包括：

- `run.created`
- `run.started`
- `message.delta`
- `message.completed`
- `tool.called`
- `tool.completed`
- `plan.generated`
- `decision.generated`
- `trace.span`
- `clarify.requested`
- `clarify.resumed`
- `checkpoint.loaded`
- `checkpoint.saved`
- `persona.selected`
- `interview_tree.generated`
- `score.generated`
- `profile.updated`
- `review.generated`
- `copilot.hint`
- `copilot.feedback`
- `run.completed`
- `run.cancelled`
- `run.failed`
- `heartbeat`

要求：

- 关键状态转移必须有结构化事件
- SSE 默认传输事件对象，不退回纯文本 chunk 协议
- review 相关 UI 优先消费 `review.generated` 或 `GET /api/runs/:id/review`

## 7. Review 标准

`ReviewSnapshot` 是复盘中心的统一读取模型，当前应继续聚合：

- `interviewState`
- `decision`
- `decisionReplay`
- `scorecard`
- `trace`
- `profile`
- `replay`
- `summary`

要求：

- 新的复盘能力优先进入 `ReviewSnapshot`
- 不要求前端自己拼多份散落事件来还原完整复盘结果

## 8. Tool 与联网标准

当前 tool plane 的重点不是“无限扩展工具市场”，而是稳定支撑 interview workflow：

- `skill.resolve`
- `rubric.resolve`
- `memory.get`
- `memory.append`
- `checkpoint.load`
- `checkpoint.save`
- artifact 读取与绑定
- 显式开关控制的 web research

要求：

- 联网检索只能在 `enableWebSearch` 或计划明确命中研究需求时触发
- tool 调用过程应产出 `tool.called / tool.completed`
- registry 可以继续扩展 MCP transport，但不能破坏本地 fallback

## 9. 前端标准

前端当前是工作台模型，不是普通聊天页。

必须保留的主路径：

- 工作区列表
- run composer
- 会话配置
- 消息流
- clarify / checkpoint / telemetry 状态展示
- review center
- replay / compare
- copilot 提示

要求：

- 前端继续围绕 `Conversation / Task / Run` 建模
- run 时间线默认基于事件流而不是字符串状态猜测
- review 组件优先消费统一快照结构

## 10. 持久化标准

### 10.1 Redis

用于：

- checkpoint
- clarify request
- 短期恢复态

### 10.2 Mongo

用于：

- conversations
- tasks
- runs
- messages
- events
- profiles
- artifact metadata

### 10.3 Artifact 文件存储

用于：

- 文本材料内容
- 上传文件原始内容

要求：

- 运行恢复所需状态不得依赖进程内 map
- artifact metadata 与 artifact payload 分离存储

## 11. 兼容与非目标

### 11.1 当前保留兼容

- `POST /api/interview` 仍存在，但属于兼容入口

### 11.2 当前未做

- GUI executor
- 广义 MCP 生态覆盖
- 多租户治理
- 通用 agent 平台化 UI

要求：

- 新功能如果超出 interview product 主链，应先确认是否真的属于本仓库范围

## 12. 文档维护规则

如果新变更会影响：

- 分层边界
- middleware 顺序
- 持久化来源
- 事件协议
- review snapshot 结构

则必须先更新本文档，再更新代码。
