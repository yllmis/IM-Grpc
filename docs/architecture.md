# IM 可观测查询 — 架构设计（Architecture）

> **本文是后续开发的单一事实来源（SSOT）中的“结构篇”。**
> 语义契约见 [message-event-contract.md](./message-event-contract.md)，
> 接口契约见 [operations-query.md](./operations-query.md)，
> 红线与闸门见 [compatibility-and-rollout.md](./compatibility-and-rollout.md)。
> 实现与本文冲突时：**先停下来，改文档或提异议，禁止私自改结构。**

---

## 0. 文档地图

```
docs/
  architecture.md                ← 本文：结构、组件、流程、决策
  message-event-contract.md      ← 事件字段 / 类型 / 脱敏 / 一致性语义
  operations-query.md            ← OperationsQuery RPC 契约 / 错误码 / 权限
  compatibility-and-rollout.md   ← 兼容红线 / 发布闸门 / 测试与验收
```

| 想查什么 | 看哪篇 |
|---|---|
| 目录放哪、依赖谁、时序如何 | 本文 |
| 事件字段叫什么、能不能存正文 | message-event-contract |
| RPC 参数、错误码、空结果语义 | operations-query |
| 什么不能改、如何上线 | compatibility-and-rollout |

---

## 1. 背景与目标

### 1.1 问题

现有 IM 链路只能回答「消息进没进 Kafka、存没存 ChatLog」，无法回答：

- 这条消息走到哪一步了？（生命周期）
- 为什么对方没收到？（投递失败 / 离线 / 无 ACK）
- 某用户在某时刻是否在线？（历史连接）
- 这些证据是否完整？（能力声明）

根因（已核实）：

| # | 现状 | 位置 |
|---|---|---|
| 1 | 客户端 `msg.Id` 不进 Kafka | `apps/im/ws/internal/handler/conversation/conversation.go` |
| 2 | Kafka 消费时重新生成 ObjectID | `apps/task/mq/internal/handler/msgTransfer/msgChatTransfer.go` |
| 3 | `sent` 只表示进入处理链路 | conversation.go 回包 |
| 4 | 投递失败 / 离线 / ACK 超时无统一持久化 | `apps/im/ws/internal/handler/push/push.go`（离线直接 return） |
| 5 | 默认 NoAck | `apps/im/ws/im.go`（`WithAck` 未启用） |
| 6 | 在线状态 ≠ 历史事实 | WS 进程内存表；Redis `online:users` 是登录态 |
| 7 | 已读只有当前 bitmap，无时间线 | `ChatLog.ReadRecords` + `pkg/bitmap` |
| 8 | `ChatLog.Status` 未暴露且未写入 | `apps/im/immodels/chatlogtypes.go` |

### 1.2 目标

为 IM 增加**通用、只读、可审计**的消息生命周期查询能力，同时：

- 旧 RPC / 旧客户端 / 旧 Kafka 流程零语义变化
- 观测失败不阻塞业务
- 不为某个 Agent 定制，服务名为 `OperationsQuery`

### 1.3 非目标

- 不做消息可靠投递改造（不改 Retry / 离线信箱）
- 不做审计合规级“零丢失”保证（Mongo 4.2 无事务）
- 不做写操作 / 管理动作
- 不依赖任何 Agent / TypeScript / Next.js 工程

---

## 2. 现状架构（As-Is）

### 2.1 组件

```
apps/
  user/     user-api(HTTP) + user-rpc(gRPC) + models(MySQL)
  social/   social-api(HTTP) + social-rpc(gRPC) + socialmodels(MySQL)
  im/       im-api(HTTP) + im-rpc(gRPC) + im-ws(WebSocket) + immodels(MongoDB)
  task/     mq(Kafka consumer → Mongo 落库 → WS 回推)
pkg/        共享库（bitmap / interceptor / xerr / wuid / constants ...）
```

### 2.2 消息主链路（现状）

```
Client ──WS──► im-ws.Chat
                 │ 生成 ConversationId
                 │ SendTime = now.UnixNano()
                 │ 丢弃客户端 msg.Id（仅回显）
                 ▼
              Kafka  MsgChatTransfer
                 │  无消息 ID 字段
                 ▼
              task-mq.Consume
                 │  msgId = bson.NewObjectID()   ← 此处才生成 ID
                 ▼
              MongoDB chat_log  (_id = msgId)
                 │
                 ▼
              task-mq → WsClient.Send("push")
                 ▼
              im-ws.push
                 ├─ GetConn(recvId) == nil → return nil   ← 离线被吞掉
                 └─ Send(socket)                         ← 成败无人记录
```

### 2.3 已读回执链路（现状，保留复用）

```
Client ──WS MarkRead──► Kafka MsgMarkRead
                          ▼
                     task-mq.UpdateChatLogRead
                       单聊: ReadRecords = []byte{1}
                       群聊: bitmap.Set(userId)
                          ▼
                     chat_log.ReadRecords（覆盖写，无时间戳）
                          ▼
                     回推 ContentReadType（群聊有合并延迟）
```

### 2.4 连接状态（现状）

| 载体 | 语义 | 问题 |
|---|---|---|
| `Server.userToConn` | 本进程 WS 连接 | 单实例内存，非历史 |
| Redis `online:users` | **登录**时 Hset，WS 关闭 Hdel | 登录态 ≠ 连接态；多实例下仅“有共享存储”但无时间线 |

---

## 3. 目标架构（To-Be）

### 3.1 总览

```
                         【业务路径：保持不变】
Client ──WS──► im-ws ──Kafka──► task-mq ──Mongo──► chat_log
                 │                  │                  │
                 │                  │                  └──► 会话未读/已读
                 │                  └──WS push──► im-ws ──WS──► Client
                 │
                 └──WS 回包 {msgId, serverMessageId, status:"sent"}

                         【观测路径：只旁路】
   im-ws / task-mq / mqclient
            │  ObservationSink.Record()
            ▼
   pkg/observation
     ├─ NoopObservationSink        ← Enabled=false（默认）
     ├─ AsyncObservationSink       ← 缓冲 + 后台写，满则丢弃计数
     └─ MongoObservationSink       ← 写 message_events
            │
            ▼
   MongoDB  message_events  ──只读──►  OperationsQuery (gRPC)
                                          │
                                          ▼
                              客服后台 / 运维 / 审计 / 监控 / 诊断
```

**原则：两条路径物理分离。** 业务路径不 import 查询服务；观测路径失败不影响业务路径返回值。

### 3.2 消息全链路目标时序（含事件）

```
Client          im-ws           Kafka         task-mq        Mongo         事件流
  │  chat(msg.Id) │                │              │             │              │
  ├──────────────►│                │              │             │              │
  │               │ gen messageId  │              │             │              │
  │               │ gen corrId     │              │             │              │
  │               │──────────────────────────────────────────────────────────►│ accepted
  │               │ publish        │              │             │              │
  │               ├───────────────►│              │             │              │
  │               │                │─────────────────────────────────────────►│ kafka_published
  │  sent{msgId,  │                │              │             │              │
  │  serverMsgId} │                │              │             │              │
  │◄──────────────┤                │              │             │              │
  │               │                │ consume      │             │              │
  │               │                ├─────────────►│             │              │
  │               │                │              │──────────────────────────►│ kafka_consumed
  │               │                │              │ insert      │              │
  │               │                │              ├────────────►│              │
  │               │                │              │──────────────────────────►│ persisted
  │               │                │              │ push(per recvId)          │
  │               │◄───────────────┼──────────────┤             │              │
  │               │ delivery       │              │             │              │
  │               │  ├ online conn │              │             │              │
  │               │  │  ─────────────────────────────────────────────────────►│ delivery_succeeded
  │               │  └ no conn     │              │             │              │
  │               │     ─────────────────────────────────────────────────────►│ receiver_offline
  │◄──────────────┤ message        │              │             │              │
```

失败分支：

| 失败点 | 事件 | 业务侧表现（不变） |
|---|---|---|
| Kafka publish | `kafka_publish_failed` | WS 回错误帧（现行为） |
| Mongo insert | `persist_failed` | 消费错误（现行为） |
| socket 写出 | `delivery_failed` | 现行为不改 |
| 无连接 | `receiver_offline` | 仍视为成功（现行为） |

### 3.3 连接事件时序

```
握手鉴权通过 ──► addConn ──► 事件 online  {connectionId, instanceId, reason=connect}
Close/顶号/超时 ──► Server.Close ──► 事件 offline {connectionId, instanceId, reason}
                                   + 现有 Redis Hdel（保留，不作为历史源）
```

连接事件与消息事件**同库同集合** `message_events`，用 `eventType=online|offline` 区分，`messageId` 留空。

### 3.4 ACK 观测时序（仅开启时）

```
readAck 循环（现有逻辑不改）
  ├─ OnlyAck：服务端发送 ACK 后结束队列 ──► 不产生 ack_received
  ├─ RigorAck：收到客户端真实 ACK ──► 事件 ack_received
  ├─ RigorAck 超时分支   ──► 事件 ack_timeout
  └─ NoAck 模式          ──► 不产生任何 ack_* 事件
```

---

## 4. 组件视图

### 4.1 新增组件

| 组件 | 路径 | 职责 |
|---|---|---|
| 观测共享库 | `pkg/observation/` | 事件类型、Sink 接口、Noop/Async/Mongo 实现、配置 |
| 事件存储模型 | `pkg/observation/mongo.go`（或 `apps/operations/operationsmodels/`，二选一，见 ADR-009） | `message_events` 读写 |
| 查询服务 | `apps/operations/rpc/` | `OperationsQuery` 六个只读 RPC |
| 事件查询模型 | `apps/operations/operationsmodels/` | 时间线/投递/连接的查询封装 |

### 4.2 `pkg/observation` 结构

```
pkg/observation/
  event.go      MessageEvent、EventType 常量、EventVersion
  sink.go       interface ObservationSink; NoopObservationSink
  async.go      AsyncObservationSink（channel 缓冲、丢弃计数、优雅退出）
  mongo.go      MongoObservationSink（SetOnInsert 幂等写）
  config.go     DeliveryObservation 配置与校验
  metrics.go    events_recorded_total / events_dropped_total / record_seconds
```

依赖方向（强制）：

```
im-ws ──────────┐
task-mq ────────┼──► pkg/observation ──► mongo-driver
mqclient ───────┘         ▲
                          │ （禁止反向依赖）
operations ───────────────┘  只读复用 EventType / MessageEvent 结构
```

**禁止** `pkg/observation` import `apps/**`（除测试）。

### 4.3 `apps/operations/rpc` 结构（go-zero 同构）

```
apps/operations/rpc/
  operations.proto
  operations/            # pb 生成物（提交入库，与 im/ user/ 一致）
  operationsclient/
  internal/
    config/config.go
    logic/               # 六个 RPC
      finduserreferencelogic.go
      getmessagerecordlogic.go
      getmessagetimelinelogic.go
      getdeliverytimelinelogic.go
      getconnectionobservationslogic.go
      getcapabilitieslogic.go
    server/operationsserver.go
    svc/servicecontext.go
    types/
  etc/dev/operations.yaml
  operations.go
```

ServiceContext 依赖（全部只读）：

```
ChatLogModel   immodels.ChatLogModel
EventModel     事件查询（message_events）
UserRpc        userclient.User        // FindUser / GetUserInfo
SocialRpc      socialclient.Social    // Groupusers（派生已读名单）
```

### 4.4 改动点（现有代码，只旁路）

| 文件 | 改动 | 事件 | 约束 |
|---|---|---|---|
| `apps/im/ws/internal/handler/conversation/conversation.go` | 生成 messageId/corrId，写 Kafka，回包加 `serverMessageId`，记 accepted | accepted | 回包保留 `msgId`+`status:"sent"` |
| `apps/task/mq/mq/mq.go` | `MsgChatTransfer` 增加三个可选 JSON 字段 | — | 旧消费者忽略新字段 |
| `apps/task/mq/mqclient/msgtransfer.go` | 包装 Push 记成败 | kafka_published / kafka_publish_failed | 失败仍返回原错误 |
| `apps/task/mq/internal/handler/msgTransfer/msgChatTransfer.go` | 优先用 messageId；钩子 | kafka_consumed / persisted / persist_failed | 无 messageId 走旧行为 |
| `apps/im/ws/internal/handler/push/push.go` | 每 receiverId 记投递事件 | delivery_* / receiver_offline | **返回值不变** |
| `apps/task/mq/internal/handler/msgTransfer/msgReadTransfer.go` | UpdateMakeRead 成功后记事件 | read_confirmed | 不改已读逻辑 |
| `apps/im/ws/websocket/server.go` | addConn/Close 记事件；readAck 钩子 | online / offline / ack_* | 不改连接表与重试 |
| `apps/im/ws/im.go` | 注入 sink、InstanceId、配置校验 | — | 默认 Enabled=false |
| `apps/task/mq/internal/svc/servicecontext.go` | 注入 sink | — | 同上 |

**不改：** 旧 proto、旧 API handler、ChatLog 字段语义、Kafka topic、推送路由逻辑、bitmap 算法。

---

## 5. 数据视图

### 5.1 集合关系

```
MongoDB (db: im)
  chat_log            既有：消息正文与已读 bitmap（业务真相）
  message_events      新增：事件流（观测证据，只增）
MySQL
  users / friends / groups / ...   既有，OperationsQuery 经 user-rpc / social-rpc 只读访问
Redis
  online:users        保留现状；仅可作 current.source=ws-login 辅助，不作历史源
```

### 5.2 ID 映射（一张表说清）

| 概念 | 生成方 | 形态 | 出现位置 |
|---|---|---|---|
| `clientMessageId` | 客户端 | 任意字符串 | WS 请求 `msg.Id`、事件、回包 `msgId` |
| `messageId` | im-ws accepted | 24hex ObjectID | Kafka、`chat_log._id`、事件、回包 `serverMessageId` |
| `correlationId` | im-ws accepted | 默认= messageId | Kafka、事件 |
| `eventId` | 各钩子 | 24hex | `message_events` 唯一键 |
| `attemptId` | 投递/ACK | 24hex 或拼接串 | 事件 metadata / 字段 |

```
clientMessageId ──(仅关联)──► messageId ──┬──► chat_log._id
                                          ├──► message_events.messageId
                                          └──► 回包 serverMessageId
```

### 5.3 message_events 形态

字段与索引见 [message-event-contract.md §4](./message-event-contract.md)。要点：

- 一事件一文档，只增不改
- `eventId` unique → 幂等
- 消息生命周期事件 `messageId` 必填；连接事件和 `observation_gap` 观测缺口事件为空
- 时间 `occurredAt` UnixNano

### 5.4 查询如何拼证据

| 问题 | 数据来源 | 能力 |
|---|---|---|
| 消息元数据是什么 | `chat_log` | message_record |
| 生命周期走到哪 | `message_events` by messageId | message_timeline |
| 对方收到没 | delivery_* by messageId+receiverId | delivery_events（gateway-write） |
| 传输层确认没 | ack_*（需开启） | ack_history（默认 unsupported） |
| 当时在线吗 | online/offline 事件 | historical_connection（默认 unsupported） |
| 业务已读没 | `chat_log.ReadRecords` 派生 + read_confirmed 事件 | read_confirmation（默认 unsupported） |
| 写入失败没 | persist_failed / kafka_publish_failed | write_failure_events（默认 unsupported） |

**证据不足时：** `truncated` / `note` / `eventsAvailable=false` / capabilities=partial，禁止用推测填充。

---

## 6. 进程与部署视图

### 6.1 进程

| 进程 | 新增职责 | 是否必须 |
|---|---|---|
| im-ws | 生成 ID、accepted/online/offline/delivery/ack 事件 | 是（改） |
| task-mq | consumed/persisted/read_confirmed 事件 | 是（改） |
| operations-rpc | 只读查询 | 是（新） |
| user-rpc / social-rpc | 被 operations 只读调用 | 否（不改） |
| im-rpc / im-api / *-api | 无 | 否（不改） |

### 6.2 部署拓扑

```
        ┌─ im-ws × N ──────┐
        │  (多实例)         │──► Redis / Mongo / Kafka
Client ─┤                  │
        └──────────────────┘

        ┌─ task-mq × N ───┐
        │  (Kafka 消费组)   │──► Mongo / WsClient
        └──────────────────┘

        ┌─ operations-rpc × N ─┐
        │  无状态               │──► Mongo(读) / user-rpc / social-rpc
        └──────────────────────┘
```

- 多实例：事件必须写共享 Mongo，`metadata.instanceId` 区分来源进程
- 无状态：查询服务可任意扩容
- 配置：Sail + etcd 可继续作为配置来源；第一版 `DeliveryObservation.Enabled` 与 `AckMode` 均在启动时读取，修改需要重启，避免 Sink 和 ACK 队列在运行中切换导致语义不一致

---

## 7. 横切关注点

### 7.1 不阻塞业务

```
ObservationSink.Record
  └─ AsyncObservationSink
       ├─ select { case ch <- ev: ; default: drop++ }
       └─ 后台 worker 批量 Mongo SetOnInsert
```

- 禁止在 WS write / Kafka produce / Mongo insert 的调用栈里同步等事件落库
- 进程退出：尽力 flush 一小段宽限期，flush 失败允许丢事件（宁丢事件不丢消息）
- 缓冲区丢弃事件时，Sink 必须累计丢弃计数和时间范围，并在后续成功写入时写入 `observation_gap` 标记；如果标记也写失败，查询只能返回 `coverageStatus=unknown`，不能返回 `complete=true`。

### 7.2 错误与超时语义

```
查询成功且无数据  →  OK + found=false / events=[]
查询未得到结果    →  DEADLINE_EXCEEDED / UNAVAILABLE / ...
```

写事件失败 → 日志 + `events_dropped_total`，不影响业务返回。

### 7.3 安全

| 层 | 策略 |
|---|---|
| 传输 | gRPC |
| 服务身份 | metadata（`x-im-service-token` 或既有机制） |
| 能力 | 只读；proto 层禁止写方法 |
| 数据 | 脱敏黑名单（事件契约 §7） |
| 多租户 | **不存在**，不虚构；边界=服务身份+网络 |

### 7.4 可观测自身的可观测

| 指标 | 用途 |
|---|---|
| `observation_events_total{type}` | 事件量 |
| `observation_events_dropped_total{reason}` | 旁路丢弃 |
| `observation_record_seconds` | 写入延迟 |
| `operations_query_total{rpc,code}` | 查询调用 |
| `operations_query_seconds{rpc}` | 查询延迟 |

---

## 8. 架构决策记录（ADR）

> 这些决策已锁定。要改必须开新 ADR 并更新全部相关文档。

### ADR-001 messageId 在 im-ws accepted 时生成
- **决策：** `bson.NewObjectID().Hex()`，贯穿 Kafka/Mongo/事件
- **理由：** 与 `FindOne` ObjectID 兼容；消费端不再生成 → 全链路可关联
- **备选否决：** UUID 字符串（要改 `_id` 类型）；消费端生成（无法关联 accepted）

### ADR-002 时间统一 int64 UnixNano
- **理由：** 与现有 `SendTime` 一致，避免双单位
- **备选否决：** `google.protobuf.Timestamp`（混用风险 > 收益）

### ADR-003 观测异步旁路，默认 Noop
- **理由：** 满足“观测失败不拖垮业务”；默认零行为变化
- **代价：** 事件可能丢；接受（见 ADR-004）

### ADR-004 事件最终一致，不做 Outbox/事务
- **理由：** 部署 Mongo 4.2 单实例，无多文档事务
- **备选否决：** 同事务 Outbox（需副本集+大改）；本期不做
- **要求：** 所有文档禁止声称强一致

### ADR-005 ACK 只观测、不改造
- **决策：** 记录现有 WS 传输层 ACK 真实发生的事；OnlyAck 不等待客户端 ACK，只有 RigorAck 收到客户端帧才记录 `ack_received`；默认 NoAck
- **理由：** 不改旧协议与重试行为；不伪造 ack
- **备选否决：** 新增业务投递 ACK 协议（旧客户端必须升级）→ 违反兼容红线

### ADR-006 `delivery_succeeded` = 网关写出成功
- **决策：** evidence=`gateway-write`
- **理由：** 现网关无法证明客户端收到；能力声明必须诚实

### ADR-007 鉴权用服务身份，不信任请求体
- **理由：** 无租户模型；避免伪造 actorId/tenantId

### ADR-008 已读复用 bitmap，不重造读回执
- **决策：** 查询侧只返回 `readState=known|unknown|approximate`；事件侧新增 `read_confirmed`；第一版不返回由 bitmap 派生的精确用户列表
- **理由：** bitmap 是业务真相；缺的是时间线而非已读本身
- **已知限制：** bitmap 为 BKDRHash，大群可能碰撞 → 派生名单仅供参考，精确名单需后续独立改造（单列 ADR）

### ADR-009 事件模型放 `pkg/observation`
- **决策：** `MessageEvent` 结构与 Mongo 写入在 `pkg/observation`；`apps/operations/operationsmodels` 只做查询封装
- **理由：** 写方（im-ws/task-mq）与读方（operations）共享契约，且写方不依赖查询服务

### ADR-010 旧消息兼容生成 ID
- **决策：** Kafka 无 `messageId` → 消费端 `bson.NewObjectID()`，事件 `source=compat-legacy`
- **理由：** 旧消息可继续入库，不破坏在途数据

---

## 9. 架构守护清单（Code Review 用）

改动是否合架构，按下列问题检查：

1. 是否改动了旧 RPC 字段语义 / `sent` 含义 / WS 协议？
2. 是否在业务同步路径上等待事件写入？
3. 是否把查询失败翻译成了 `found=false` 或空列表？
4. 是否在事件/响应中放入正文、Token、手机号、原始 bitmap？
5. 是否引入了写接口、SQL/Shell、Agent 专用命名？
6. 是否在消费端重新生成了新消息的 messageId？
7. 是否伪造了 ACK 或把 `delivery_succeeded` 写成“对方已收到”？
8. 是否虚构了租户隔离或把 partial 能力标成 supported？
9. `pkg/observation` 是否反向依赖了 `apps/*` 业务？
10. 三份契约文档是否与实现同步？

**任一项为“是” → 先改设计或改文档，再合码。**

---

## 10. 演进路线（结构视角）

```
闸门 A  契约落地：proto + message_events + messageId 贯通
         └─ 结构首次固定：ADR-001/002/009/010 落地
闸门 B  旁路钩子：Noop/Async/Mongo + 五类事件
         └─ ADR-003/004/006/008 落地
闸门 C  连接事件 + ACK 观测
         └─ ADR-005 落地
闸门 D  OperationsQuery 六 RPC
         └─ ADR-007 落地
闸门 E  测试 / 文档核对 / 全量门禁
```

后续可选演进（本期不做，勿提前设计）：

- Outbox + Mongo 副本集事务（推翻 ADR-004 需新 ADR）
- 精确已读名单（bitmap 碰撞治理）
- HTTP/REST 网关、gRPC-gateway
- 事件流订阅（Kafka 转储 message_events）
- 消息可靠投递 / 离线信箱

---

## 11. 术语表

| 术语 | 含义 |
|---|---|
| 业务路径 | 发消息、落库、推送的主流程 |
| 观测路径 | 旁路记录事件的流程 |
| gateway-write | 网关 socket 写出成功（非客户端送达） |
| transport-ack | WS 传输层 ACK |
| read_confirmed | 业务已读回执入库成功 |
| compat-legacy | 旧格式消息的兼容处理 |
| 证据不足 | 查询成功但无法给出结论（≠ 查不到数据） |
| sent | 消息进入 Kafka 处理链路（≠ delivered） |
