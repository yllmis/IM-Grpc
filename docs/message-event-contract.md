# 消息事件契约（Message Event Contract）

> 本文档是观测事件的**唯一契约来源**。实现、测试、文档、能力声明都必须与本文一致。
> 发现与代码不一致时，先改文档或先提异议，禁止“顺手改语义”。

相关文档：[architecture.md](./architecture.md) · [operations-query.md](./operations-query.md) · [compatibility-and-rollout.md](./compatibility-and-rollout.md)

---

## 1. 目标与非目标

### 1.1 目标

为消息全生命周期提供**只读、可审计、可关联**的事件证据，支撑客服后台、运维排障、审计、监控、外部诊断等系统。

### 1.2 非目标

| 非目标 | 说明 |
|---|---|
| 不保证强一致 | 事件与业务写入是最终一致，见 §8 |
| 不替代 ChatLog | 事件是证据流，不是消息正文存储 |
| 不承载消息正文 | 禁止写入 `msgContent` 等，见 §7 |
| 不改变投递/ACK 行为 | 只旁路记录，不参与业务决策 |

---

## 2. 时间格式（强制）

**全链路统一使用 `int64` UnixNano（纳秒）。**

| 规则 | 说明 |
|---|---|
| 单位 | 纳秒，UTC 时间戳 `time.Now().UnixNano()` |
| proto 字段命名 | `xxxAt` / `xxxTime` 一律 `int64`，注释标注 `// UnixNano` |
| 禁止 | 毫秒、秒、`google.protobuf.Timestamp` 混用 |
| 兼容既有 | 与 `ChatLog.SendTime`、`mq.MsgChatTransfer.SendTime` 同单位 |

> 选择 UnixNano 而非 `Timestamp` 的原因：项目现有时间字段全部为 UnixNano，引入 Timestamp 会造成同一响应内双单位，违反“不能混用”要求。若未来引入 Timestamp，必须一次性全量迁移并升 `eventVersion`。

非法时间：`startTime > endTime`、时间为负、超出 `[2000-01-01, now+1h]` 合理区间 → `INVALID_ARGUMENT`。

---

## 3. 标识符体系

三个 ID 各司其职，**禁止互相重新解释**。

| ID | 生成方 | 生成时机 | 生命周期 | 说明 |
|---|---|---|---|---|
| `clientMessageId` | 客户端 | 发消息时 | 透传 | 即 WS 协议里的 `msg.Id`，仅用于回显与请求关联 |
| `messageId` | **服务端 im-ws** | 消息 `accepted` 时 | 全链路稳定 | `bson.NewObjectID().Hex()`，贯穿 Kafka / Mongo / 事件 / ACK |
| `correlationId` | 服务端 im-ws | 与 messageId 同时 | 全链路透传 | 请求追踪 ID，可选；缺省等于 messageId |
| `eventId` | 写事件方 | 记录事件时 | 事件唯一 | `bson.NewObjectID().Hex()`，幂等主键 |
| `attemptId` | 投递/ACK 逻辑 | 每次投递尝试 | 单次尝试 | 区分同一消息的多次投递/重试 |

### 3.1 硬性约束

1. **新消息**必须使用进入链路时生成的 `messageId`，Kafka 消费端**禁止重新生成**。
2. **旧消息**（Kafka payload 无 `messageId` 字段）消费端按旧行为 `bson.NewObjectID()` 生成，并写入事件时 `clientMessageId` 为空、`source` 标注 `compat-legacy`。
3. `clientMessageId` **不得**直接充当数据库 `_id`。
4. WS 回包保留 `msgId`（= clientMessageId），新增 `serverMessageId`（= messageId），二者并存。

### 3.2 Kafka 消息扩展（向后兼容）

在 `mq.MsgChatTransfer` 上增加**可选** JSON 字段，旧消费者 `json.Unmarshal` 自动忽略：

```json
{
  "messageId": "665f1c...24hex",
  "clientMessageId": "client-uuid-or-seq",
  "correlationId": "req-...",

  "conversationId": "...",
  "sendId": "...",
  "recvId": "...",
  "chatType": 2,
  "sendTime": 1727500000000000000,
  "mType": 0,
  "content": "..."
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `messageId` | 新消息必填，旧消息可空 | 24 位 hex ObjectID |
| `clientMessageId` | 可空 | 客户端原始 msg.Id |
| `correlationId` | 可空 | 链路追踪 |
| 其余字段 | 保持原语义 | 不删、不改义 |

---

## 4. 事件模型

### 4.1 存储

- MongoDB 集合：`message_events`（新建，与 `chat_log` 同库）
- 文档形态：单条事件一个文档
- 集合生命周期：只增不改（除 `_id` 索引维护）；**禁止**事件更新或删除 API

### 4.2 字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `eventId` | string | 是 | 唯一，幂等键 |
| `eventVersion` | int32 | 是 | 当前 `1`；语义变更时递增 |
| `eventType` | string | 是 | 见 §5 |
| `messageId` | string | 消息生命周期事件必填；连接/观测缺口事件为空 | 服务端消息 ID |
| `clientMessageId` | string | 可空 | 客户端 msg.Id |
| `correlationId` | string | 可空 | 追踪 ID |
| `conversationId` | string | 可空 | 会话 ID |
| `senderId` | string | 可空 | 发送方；**连接事件=被观察用户** |
| `receiverId` | string | 可空 | 接收方（群聊按接收方拆多条） |
| `attemptId` | string | 可空 | 投递/重试尝试 |
| `occurredAt` | int64 | 是 | UnixNano，事件发生时刻 |
| `source` | string | 是 | 产生方，见 §6 |
| `errorCode` | string | 可空 | 失败事件填错误码；成功为空 |
| `sequence` | int64 | 可空 | 同消息内单调序号（尽力而为） |
| `metadata` | map[string]string | 可空 | 轻量键值；禁止敏感数据 |

### 4.3 索引（必须）

| 索引 | 类型 | 用途 |
|---|---|---|
| `eventId` | **unique** | 幂等去重 |
| `messageId + occurredAt` | 复合 | 消息时间线 |
| `correlationId` | 单键 | 请求追踪 |
| `receiverId + occurredAt` | 复合 | 投递时间线、连接观察 |
| `eventType + occurredAt` | 复合 | 按类型审计 |

### 4.4 幂等语义

```
同一 eventId 重复写入 → 必须不产生第二条记录
```

实现要求：

1. 以 `eventId` unique 索引兜底（并发下唯一约束是最终防线）。
2. 写入前可先用 eventId 做 upsert / `UpdateOne(..., SetOnInsert)`，降低冲突报错噪音。
3. 重复写入**不是错误**，记 debug 日志即可，不得影响业务。
4. 事件写入失败：记错误日志 + 指标，**不得**重试阻塞业务主路径；不得回滚消息。

---

## 5. 事件类型

### 5.1 消息生命周期（必选实现）

| eventType | 触发点 | 含义 | 成功/失败 |
|---|---|---|---|
| `accepted` | im-ws 收到合法 chat 消息、即将投递 Kafka | 消息被接入层接受 | 成功 |
| `kafka_published` | Kafka push 成功 | 消息已发布到 Kafka | 成功 |
| `kafka_publish_failed` | Kafka push 失败 | 发布失败（业务已回错误给客户端） | 失败 + errorCode |
| `kafka_consumed` | task-mq 成功反序列化并开始处理 | 消费开始 | 成功 |
| `persisted` | ChatLog + 会话更新成功 | 消息已持久化 | 成功 |
| `persist_failed` | ChatLog 写入/更新失败 | 持久化失败 | 失败 + errorCode |
| `delivery_attempted` | 开始向某个接收方推送 | 一次投递尝试开始 | 中性 |
| `delivery_succeeded` | 网关向该接收方 socket **写出成功** | 见 §5.3 语义边界 | 成功 |
| `delivery_failed` | socket 写出失败 / 推送异常 | 投递失败 | 失败 + errorCode |
| `receiver_offline` | 接收方无 WebSocket 连接 | 当前不可达 | 中性 |
| `ack_received` | 收到**真实**传输层 ACK | 见 §5.4 | 成功 |
| `ack_timeout` | 超过真实 ACK 超时阈值且未收到 ACK | 见 §5.4 | 失败 |
| `read_confirmed` | 已读回执入库成功（`UpdateMakeRead`） | 业务已读成立 | 成功 |

### 5.2 连接观察（必选实现）

| eventType | 触发点 | 含义 |
|---|---|---|
| `online` | WS 连接鉴权通过、进入连接表 | 连接建立 |
| `offline` | WS 连接关闭（含顶号、超时、主动断开） | 连接断开 |

当异步 Sink 因缓冲区满、进程退出或后端不可用而丢弃事件时，必须尽力写入一个 `observation_gap` 标记。该标记的 `metadata` 至少包含 `droppedCount`、`gapStartAt`、`gapEndAt` 和 `reason`；它不是业务消息事实，只用于告诉查询侧该时间窗口的观测覆盖不完整，`messageId` 可以为空。

连接事件字段约定：

- `messageId` 留空
- `senderId` = 被观察 userId
- `metadata.connectionId`、`metadata.instanceId`、`metadata.reason`
- `state` 由 eventType 决定（`online` / `offline`），不再单独存 state 字段

### 5.3 `delivery_succeeded` 语义边界（必须写进所有文档）

```
delivery_succeeded == 网关进程向接收方 WebSocket socket 写出成功
```

**不代表：**

- 客户端已收到
- 客户端已渲染
- 客户端已确认
- 端到端送达

证据强度：`gateway-write`。查询响应中的 `source` 必须如实标注，禁止在 UI/文档把它表述为“对方已收到”。

群聊：`RecvIds` 中**每个接收方独立**产生 `delivery_*` / `receiver_offline` 事件，`receiverId` 为具体用户，禁止合并为一条。

### 5.4 ACK 事件语义

观测对象是 **WebSocket 传输层 ACK**（`NoAck / OnlyAck / RigorAck`），不是业务已读。

| 规则 | 说明 |
|---|---|
| NoAck 模式 | **不写** `ack_received` / `ack_timeout`，禁止伪造 |
| OnlyAck 模式 | 服务端发送 ACK 后即移除/结束队列，不等待客户端 ACK；**不能**写 `ack_received` 或 `ack_timeout`（如需观测服务端 ACK，可另定义 `ack_sent`，本期不要求） |
| RigorAck 模式 | 只有收到客户端真实 ACK 帧时才写 `ack_received` |
| `ack_timeout` | 仅在 RigorAck 超时分支触发时写；禁止用定时器猜测 |
| 不改行为 | `readAck` 重试、退避、丢弃策略禁止因观测而改变 |
| 与已读区分 | 业务已读是 `read_confirmed`，禁止用 `ack_received` 冒充已读 |

默认配置下 ACK 事件为 0 条，`GetCapabilities.ack_history` 必须为 `unsupported`；启用 RigorAck 且确有事件后才可声明 `partial` 或 `supported`，禁止虚报。

### 5.5 可扩展

- eventType 允许追加；查询端遇到未知类型必须原样返回，禁止报错。
- 删除或改变既有 eventType 含义 → 必须递增 `eventVersion` 并更新三份文档。

---

## 6. source 取值

| source | 产生方 |
|---|---|
| `im-ws` | WebSocket 接入层 |
| `task-mq` | Kafka 消费者 |
| `mq-push` | Kafka 生产侧（可能与 im-ws 同进程） |
| `ws-push` | WebSocket 投递侧 |
| `operations` | 仅查询，不产生事件 |
| `compat-legacy` | 旧消息兼容路径 |

---

## 7. 数据脱敏（强制黑名单）

事件中**禁止**出现：

| 禁止项 | 说明 |
|---|---|
| 密码、Token、Cookie、JWT | 任何形式 |
| 完整消息正文（msgContent / content） | 即使加密后也不行 |
| 完整手机号 | 可存脱敏哈希或后 4 位，禁止全号 |
| 数据库连接串、凭证 | |
| Kafka 原始 payload | 禁止整包入 metadata |
| Redis 原始值 | |
| readRecords 原始 bitmap | 只允许派生后的 userId 列表 |
| 客户端 IP 全量长期存 | 如需保留，必须评估合规并写入本文 |

`metadata` 只允许：连接 ID、实例 ID、尝试次数、错误码、有限枚举、截断后的短标识（≤128 字符）。

违反脱敏的测试必须失败（见 compatibility 文档测试清单）。

---

## 8. 一致性与可靠性

### 8.1 写入时序（默认）

```
业务操作成功（回包 / 入库 / 投递）
        ↓
异步记录观测事件（ObservationSink）
        ↓
失败 → 日志 + 指标，不影响业务
```

### 8.2 禁止

- 禁止为了写事件阻塞 Kafka produce、WS send、Mongo insert。
- 禁止事件失败导致消息发送失败。
- 禁止查询侧伪造缺失事件（空就是空）。

### 8.3 事务现状

| 项 | 现状 |
|---|---|
| MongoDB 版本（docker-compose） | 4.2 单实例 → **无多文档事务** |
| ChatLog + 事件同事务 | **不支持** |
| Outbox | 本期不实现 |
| 保证级别 | **最终一致**；事件可能丢失、可能乱序到达 |

文档、接口注释、能力声明均不得声称“事件与 ChatLog 绝对同时成功”。

### 8.4 查询侧约定

- 事件缺失 = 证据不足，不是消息不存在。
- `complete=false` / `truncated=true` 表示时间线被截断或观测窗口未覆盖。
- 异步缓冲区丢弃事件时，`AsyncObservationSink` 必须记录可查询的 `observation_gap` 标记（至少包含丢弃计数和时间范围）；如果连缺口标记也无法写入，查询侧必须将覆盖状态标为 `unknown`。
- `coverageStatus` 只能是 `complete`、`partial`、`unknown`。只要目标窗口存在观测缺口，就禁止返回 `complete=true`；空事件列表在 `partial/unknown` 时只能表示证据不足，不能表示业务事件不存在。
- 观测启用之前的时段，查询结果可能为空 → 必须能在响应中体现（见 operations-query.md 的 `complete` 与 capabilities）。

---

## 9. ObservationSink 接口

放置位置：`pkg/observation/`（共享包，**禁止**让 im-ws / task-mq 依赖 operations 服务代码）。

```go
type MessageEvent struct {
    EventID         string
    EventVersion    int32
    EventType       string
    MessageID       string
    ClientMessageID string
    CorrelationID   string
    ConversationID  string
    SenderID        string
    ReceiverID      string
    AttemptID       string
    OccurredAt      int64 // UnixNano
    Source          string
    ErrorCode       string
    Sequence        int64
    Metadata        map[string]string
}

type ObservationSink interface {
    Record(ctx context.Context, event MessageEvent) error
}
```

### 9.1 实现

| 实现 | 行为 | 使用场景 |
|---|---|---|
| `NoopObservationSink` | 直接返回 nil | 默认、观测关闭、单测 |
| `MongoObservationSink` | 异步写 `message_events` | 测试/生产开启观测后 |
| `AsyncObservationSink` | 包装任意 Sink，缓冲后后台写，满则丢弃+计数 | **默认包装层**，保证不阻塞 |

### 9.2 钩子挂载点（只旁路，不改语义）

| 挂载点 | 文件 | 事件 |
|---|---|---|
| WS chat 接受 | `apps/im/ws/internal/handler/conversation/conversation.go` | `accepted` |
| Kafka 生产 | `apps/task/mq/mqclient/msgtransfer.go` | `kafka_published` / `kafka_publish_failed` |
| Kafka 消费 | `apps/task/mq/internal/handler/msgTransfer/msgChatTransfer.go` | `kafka_consumed` |
| Mongo 持久化 | 同上 `addChatLog` | `persisted` / `persist_failed` |
| WS 投递 | `apps/im/ws/internal/handler/push/push.go` | `delivery_*` / `receiver_offline` |
| 已读入库 | `apps/task/mq/internal/handler/msgTransfer/msgReadTransfer.go` | `read_confirmed` |
| WS 连接/断开 | `apps/im/ws/websocket/server.go` + `apps/im/ws/im.go` | `online` / `offline` |
| 传输 ACK | `apps/im/ws/websocket/server.go` `readAck` | `ack_received` / `ack_timeout`（仅开启时） |

### 9.3 配置

```yaml
DeliveryObservation:
  Enabled: false        # 总开关，false=NoopSink
  PersistEvents: true   # Enabled=true 时是否写 Mongo
  AckMode: NoAck        # NoAck | OnlyAck | RigorAck；与 WS AckType 同枚举
  InstanceId: ""        # 多实例标识；空则 hostname
  BufferSize: 1024      # 异步缓冲
```

**默认值禁止改变现有行为**：`Enabled: false` + `AckMode: NoAck`。

测试环境可开：

```yaml
DeliveryObservation:
  Enabled: true
  PersistEvents: true
  AckMode: RigorAck
```

`AckMode` 只影响**观测与 WS AckType 配置**，禁止在代码里偷偷改默认 AckType；WS 的 `WithAck` 仍由自身配置显式控制，二者必须一致（启动时校验，不一致则拒绝启动）。

---

## 10. 错误码（事件内）

| errorCode | 场景 |
|---|---|
| `KAFKA_PUBLISH_FAILED` | Kafka 发布失败 |
| `PERSIST_FAILED` | Mongo 写入失败 |
| `DELIVERY_WRITE_FAILED` | socket 写出失败 |
| `DELIVERY_NO_CONN` | 无连接（同时用 eventType=`receiver_offline`） |
| `ACK_TIMEOUT` | 传输 ACK 超时 |
| `INVALID_EVENT` | 事件自身非法被拒收（本地丢弃） |

事件内的 `errorCode` 是**业务枚举**，与 gRPC 状态码无关。gRPC 状态码见 operations-query.md。

---

## 11. 能力声明基线

`GetCapabilities` 必须与实际实现一致。各能力判定标准：

| capability | `supported` 条件 | 默认可达状态 |
|---|---|---|
| `message_record` | 可按 messageId 查 ChatLog 且区分空/错 | `supported` |
| `message_timeline` | message_events 时间线查询可用且覆盖状态可证明 | `unsupported`（默认观测关闭） |
| `delivery_events` | delivery_* 事件已落库且覆盖状态可证明 | `unsupported`（默认观测关闭） |
| `historical_connection` | online/offline 可按时间查且观测已启用 | `unsupported`（默认关闭，不回填） |
| `ack_history` | ack_* 事件可查 | `unsupported`（默认 NoAck 不产生事件） |
| `write_failure_events` | persist_failed / kafka_publish_failed 可查 | `unsupported`（默认观测关闭） |
| `read_confirmation` | read_confirmed 可查 | `unsupported`（默认关闭，不回填） |

取值仅限：`supported` / `partial` / `unsupported`。
**禁止为了让接口好看而把 partial 写成 supported。**

---

## 12. 变更流程

1. 改本文契约（含 eventType、字段、语义）。
2. 同步 operations-query.md 与 compatibility-and-rollout.md。
3. 更新单测（契约测试必须覆盖变更点）。
4. 若破坏语义 → `eventVersion` + 1，并在 compatibility 文档写迁移说明。
