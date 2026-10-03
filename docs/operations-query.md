# OperationsQuery 服务能力（只读观测查询）

> 本文档定义 `OperationsQuery` 服务的**架构、接口契约、错误语义、权限边界**。
> 实现必须与本文一致；跑偏时以本文为准。

相关文档：[architecture.md](./architecture.md) · [message-event-contract.md](./message-event-contract.md) · [compatibility-and-rollout.md](./compatibility-and-rollout.md)

---

## 1. 定位

### 1.1 是什么

IM 系统的**通用只读运维/观测查询服务**。面向：

- 客服后台排障
- 运维排障系统
- 审计系统
- 监控/告警系统
- 外部诊断服务
- 未来的 Agent Connector

### 1.2 不是什么

| 非目标 | 说明 |
|---|---|
| 不是 Agent 专用接口 | 禁止 `AgentQuery` / `DiagnosisQuery` 命名 |
| 不是消息业务接口 | 发消息、改消息走既有链路 |
| 不是管理后台写接口 | 无任何写/执行能力 |
| 不是 SQL/Shell 入口 | 永久禁止 |

### 1.3 命名

服务名：`OperationsQuery`
proto package：`operations`
目录：`apps/operations/rpc/`

---

## 2. 架构

### 2.1 在现有链路中的位置

```
┌──────────┐   WS    ┌─────────┐  Kafka  ┌──────────┐  WS   ┌──────────┐
│ Client   │ ──────► │  im-ws  │ ──────► │ task-mq  │ ────► │  im-ws   │ ──► Client
└──────────┘         └────┬────┘         └────┬─────┘       └──────────┘
                          │ accepted          │ consumed
                          │ kafka_published   │ persisted
                          │ online/offline    │ read_confirmed
                          │ delivery_*        │
                          ▼                   ▼
                    ┌────────────────────────────────────┐
                    │  ObservationSink (pkg/observation) │
                    │  Noop / Async / Mongo              │
                    └────────────────┬───────────────────┘
                                     │ 异步写（不阻塞业务）
                                     ▼
                          ┌─────────────────────┐
                          │ MongoDB             │
                          │  chat_log           │  ◄── 既有消息存储
                          │  message_events     │  ◄── 新增事件流
                          └──────────┬──────────┘
                                     │ 只读查询
                                     ▼
                          ┌─────────────────────┐
                          │  OperationsQuery    │  apps/operations/rpc
                          │  (gRPC, read-only)  │
                          └──────────┬──────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    ▼                ▼                ▼
               客服后台          运维/审计         诊断/Agent
```

### 2.2 代码布局

```
pkg/observation/                  # 共享观测包（无业务依赖）
  event.go                        # MessageEvent + eventType 常量
  sink.go                         # ObservationSink + Noop
  async.go                        # AsyncObservationSink（不阻塞）
  mongo.go                        # MongoObservationSink
  config.go                       # DeliveryObservation 配置

apps/operations/rpc/
  operations.proto                # 服务契约（本文 §4）
  operations/                     # 生成的 pb（手工提交，与现有 im/ user/ 一致）
  operationsclient/               # 客户端封装
  internal/
    logic/                        # 六个 RPC 的实现
    server/                       # gRPC server 注册
    svc/                          # ServiceContext（ChatLogModel / EventModel / User / Social）
    types/                        # 内部类型
  etc/dev/operations.yaml
  operations.go                   # main

docs/
  operations-query.md             # 本文
  message-event-contract.md
  compatibility-and-rollout.md
```

### 2.3 依赖约束（强制）

| 允许 | 禁止 |
|---|---|
| `pkg/observation`、`pkg/xerr`、`pkg/interceptor` | 依赖 frontend / 任何 TS/Next.js |
| `apps/im/immodels`（读 ChatLog） | 依赖“另一个 Agent 项目”才能编译 |
| user-rpc / social-rpc（只读查用户/群成员） | 写 ChatLog / 写会话 |
| MongoDB / Redis 只读 | 写 Kafka、发 WS |

`pkg/observation` **不得** import `apps/operations/*`（写侧不能依赖查询服务）。

### 2.4 部署形态

- 独立 gRPC 服务进程，与 im-rpc 同级
- 配置来源可沿用既有 `configserver`（Sail + etcd），但第一版 `DeliveryObservation.Enabled` 与 `AckMode` 只在进程启动时读取，修改必须重启；不把热更新能力写成已实现保证
- 不需要 HTTP 网关；需要 REST 的调用方走 gRPC-gateway 或自建 BFF（本期不做）
- 多实例无状态，可水平扩展

---

## 3. 通用约定

### 3.1 时间

一律 `int64` UnixNano（纳秒），与事件契约 §2 相同。proto 注释必须写 `// UnixNano`。

### 3.2 分页

| 项 | 约定 |
|---|---|
| `limit` | 默认 50；最大 200；`limit<=0` 用默认值；`limit>200` → `INVALID_ARGUMENT` |
| `cursor` | 不透明字符串；空=第一页 |
| 游标编码 | base64(`{occurredAt}:{eventId}`) 由服务端生成，客户端禁止解析 |
| 排序 | `occurredAt` 升序；同毫秒用 `eventId` 决胜 |
| `nextCursor` | 有下一页时非空；否则空串 |
| `complete` | `true` 表示该查询窗口内已取尽 |
| `truncated` | `true` 表示因 limit/窗口被截断，证据不完整 |

### 3.3 空结果 vs 错误（强制）

```
查询成功但没数据  →  正常响应（found=false / events=[]）
查询未获得结果    →  gRPC 错误（DEADLINE_EXCEEDED / UNAVAILABLE / ...）
```

**禁止**把超时、连接失败、权限错误翻译成 `found=false` 或空列表。
**禁止**把 `found=false` 当错误返回。

### 3.4 gRPC 错误码（强制）

| 场景 | gRPC 状态 | 说明 |
|---|---|---|
| 参数非法（空 messageId、limit 越界、startTime>endTime、坏 cursor） | `INVALID_ARGUMENT` | 响应体给出字段级原因 |
| 无权限 / 服务身份不合法 | `PERMISSION_DENIED` | 不泄露资源是否存在 |
| 查询超时 | `DEADLINE_EXCEEDED` | 含客户端 deadline 与服务端预算耗尽 |
| 数据库不可用 | `UNAVAILABLE` | 连接失败、拓扑不可用 |
| 能力未启用/不支持 | `UNIMPLEMENTED` | 如查询 ack 但 ack_history=unsupported |
| 其它内部错误 | `INTERNAL` | 不暴露堆栈 |

自定义 `xerr` 必须在服务端 interceptor 中映射为上述状态（复用 `pkg/interceptor/rpcserver` 模式）。

### 3.5 鉴权

- 传输层：gRPC
- 身份：**服务身份**（复用/扩展既有服务间认证，如 system token / interceptor），放在 **gRPC metadata**
- **禁止**把请求体中的 `actorId` / `tenantId` / `permission` 当作可信身份
- 当前系统**无租户模型** → 不虚构租户隔离；边界=服务身份 + 网络可达 + 只读
- 权限不足一律 `PERMISSION_DENIED`，不区分“没权限”和“不存在”（防探测）

服务端必须在 `OperationsQuery` gRPC interceptor 中执行认证，而不是交给业务 logic 自行判断：

1. 从 metadata 读取 `x-im-service-token`（或已批准的等价机制）；
2. 校验签发来源、有效期和允许的服务范围；无 token、过期或校验失败统一返回 `PERMISSION_DENIED`；
3. token 由部署配置/密钥系统签发和轮换，本期不在请求体中接受 actor、tenant 或 permission 声明；
4. token、Authorization metadata 和校验详情不得写入日志、Trace 或事件；
5. 轮换期间允许配置新旧 token 的短暂重叠窗口，过期后旧 token 必须失效。

### 3.6 只读红线

永久禁止出现在 `OperationsQuery` 的能力：

```
ExecuteSQL / ExecuteShell / ResendMessage / ModifyMessage
DeleteMessage / KickUser / SubmitIncident / 任意写接口
```

代码评审硬性检查项。新增 RPC 必须是只读查询，且在本文档登记。

---

## 4. 服务契约

```protobuf
syntax = "proto3";
package operations;
option go_package = "./operations";

service OperationsQuery {
  rpc SearchMessages(SearchMessagesRequest) returns (SearchMessagesResponse);
  rpc FindUserReference(FindUserReferenceRequest) returns (FindUserReferenceResponse);
  rpc GetMessageRecord(GetMessageRecordRequest) returns (GetMessageRecordResponse);
  rpc GetMessageTimeline(GetMessageTimelineRequest) returns (GetMessageTimelineResponse);
  rpc GetDeliveryTimeline(GetDeliveryTimelineRequest) returns (GetDeliveryTimelineResponse);
  rpc GetConnectionObservations(GetConnectionObservationsRequest) returns (GetConnectionObservationsResponse);
  rpc GetCapabilities(GetCapabilitiesRequest) returns (GetCapabilitiesResponse);
}
```

### 4.0 SearchMessages

当客服只有发送方 `senderId` 和时间范围、尚未知道精确 `messageId` 时，用该只读 RPC
返回有限的消息候选。它只做定位，不代表消息已投递或接收。

```protobuf
message SearchMessagesRequest {
  string senderId = 1;       // 必填
  string receiverId = 2;     // 可选
  int64 startTime = 3;       // UnixNano，闭区间，最长 7 天
  int64 endTime = 4;
  int32 limit = 5;            // 默认 10，最多 20
}

message MessageReference {
  string messageId = 1;
  string conversationId = 2;
  string senderId = 3;
  string receiverId = 4;
  int64 createdAt = 5;        // UnixNano
}

message SearchMessagesResponse {
  repeated MessageReference messages = 1;
  bool truncated = 2;
  int64 observedAt = 3;       // UnixNano
}
```

`messages=[]` 是查询成功但没有候选；`truncated=true` 表示候选不完整，不能自动选择一条。
超时、数据库不可用、权限错误和未实现必须使用 §3.4 的 gRPC 错误状态，不能伪装成空数组。
该 RPC 不返回正文、密码、Token 或 Mongo 内部字段。

部署前请由运维显式创建覆盖查询条件和排序的索引（不要在 Agent 请求中自动建索引）：

```javascript
db.chat_log.createIndex(
  { sendId: 1, recvId: 1, sendTime: 1, _id: 1 },
  { name: "ops_sender_time_id" }
)
```

如果实例暂时不能建立该索引，服务仍会执行有界查询，但应在上线检查中记录性能风险；
索引创建不属于 OperationsQuery 的运行时写能力。

以下 `int64` 时间字段均为 **UnixNano**。

---

### 4.1 FindUserReference

排障用最小用户引用。**禁止**默认返回 password / token / phone / avatar / 完整 UserEntity。

```protobuf
message FindUserReferenceRequest {
  string userId   = 1;  // 精确用户 ID
  string nickname = 2;  // 昵称匹配
  string phone    = 3;  // 可选；仅精确匹配；响应中不回显完整号码
  int32  limit    = 4;
}

message UserReference {
  string userId      = 1;
  string displayName = 2;  // 昵称
  int32  status      = 3;  // 与 user.UserEntity.status 同义
  int64  observedAt  = 4;  // 本次查询观测时间 UnixNano
}

message FindUserReferenceResponse {
  repeated UserReference users = 1;
}
```

| 规则 | 说明 |
|---|---|
| 多命中 | 昵称匹配到多个 → **全部返回**，禁止自动挑选 |
| 零命中 | `users=[]`，正常返回（不是错误） |
| 参数全空 | `INVALID_ARGUMENT` |
| phone 入参 | 可用于查找，但 `UserReference` **不包含** phone 字段 |
| limit | 同 §3.2 |

---

### 4.2 GetMessageRecord

单条消息的元数据摘要（不含正文）。

```protobuf
message GetMessageRecordRequest {
  string messageId = 1;  // 服务端 messageId（24hex）；必填
}

message GetMessageRecordResponse {
  bool   found            = 1;
  string messageId        = 2;
  string conversationId   = 3;
  string senderId         = 4;
  string receiverId       = 5;
  int64  createdAt        = 6;  // 业务发送时间 UnixNano（ChatLog.SendTime）
  string source           = 7;  // 记录来源枚举，见下表
  int64  observedAt       = 8;  // 本次查询观测时间 UnixNano
  int32  chatType         = 9;
  int32  msgType          = 10;
  // 已读状态摘要；不把 bitmap 哈希碰撞伪装成精确用户名单
  string readState         = 11; // known | unknown | approximate
  string readStateNote     = 12;
  // 证据完整性
  bool   eventsAvailable  = 13; // message_events 是否已有该消息事件
  string note             = 14; // 如 "legacy-message-no-events"
}
```

**不返回：** `msgContent`、`readRecords` 原始 bitmap、MongoDB 内部字段、原始日志、Token/密码。

| `found` | HTTP/gRPC | 含义 |
|---|---|---|
| `true` | OK | ChatLog 存在 |
| `false` | OK | **查询成功**且无记录 |
| — | `DEADLINE_EXCEEDED` | 没拿到结果，**不得**当成 found=false |
| — | `UNAVAILABLE` | 存储不可用 |
| — | `INVALID_ARGUMENT` | messageId 空/非 24hex |
| — | `PERMISSION_DENIED` | 服务身份不合法 |

`source` 取值：

| 值 | 含义 |
|---|---|
| `chat_log` | 常规持久化消息 |
| `compat-legacy` | 旧链路消息（消费端生成 ID） |
| `events-only` | 仅有事件、ChatLog 未找到（异常态，found=false 且 eventsAvailable=true） |

`readState` 语义：
- `known`：只表示当前记录能确定已读状态，不代表有完整用户名单；
- `approximate`：由群聊 bitmap 哈希派生，可能碰撞，不能作为精确诊断事实；
- `unknown`：缺少成员集合、历史记录或能力未启用。

第一版不返回 `readBy/unreadBy` 用户列表，也不导出原始 bitmap；需要精确名单时必须另立数据契约。

---

### 4.3 GetMessageTimeline

某条消息的生命周期事件流。

```protobuf
message GetMessageTimelineRequest {
  string messageId = 1;  // 必填
  int64  startTime = 2;  // 可选，0=不限
  int64  endTime   = 3;  // 可选，0=不限
  int32  limit     = 4;
  string cursor    = 5;
}

message MessageEvent {
  string eventId           = 1;
  int32  eventVersion      = 2;
  string eventType         = 3;  // 原样返回，未知类型不报错
  string messageId         = 4;
  string clientMessageId   = 5;
  string correlationId     = 6;
  string conversationId    = 7;
  string senderId          = 8;
  string receiverId        = 9;
  string attemptId         = 10;
  int64  occurredAt        = 11; // UnixNano
  string source            = 12;
  string errorCode         = 13;
  int64  sequence          = 14;
  map<string, string> metadata = 15;
}

message GetMessageTimelineResponse {
  repeated MessageEvent events   = 1;
  bool   complete    = 2;
  bool   truncated   = 3;
  string nextCursor  = 4;
  string messageId   = 5;  // 回显
  string coverageStatus = 6; // complete | partial | unknown
  uint64 eventsDropped = 7;  // 可观测缺口标记汇总；未知时为 0 且 coverageStatus=unknown
}
```

| 规则 | 说明 |
|---|---|
| 过滤 | `eventType` 不过滤（时间线要完整）；连接事件和 `observation_gap` 不含此 messageId，但缺口时间范围必须参与 coverageStatus 计算 |
| 空结果 | 仅当 `coverageStatus=complete` 时，`events=[]` + `complete=true` 才能表示窗口内没有已记录事件；`partial/unknown` 只能表示证据不足 |
| 证据不足 | 若观测窗口未覆盖消息创建时间，`truncated=true` 或在 `metadata` 标注 `window_gap=true` |
| 观测丢失 | 存在 `observation_gap` 时必须 `complete=false`、`coverageStatus=partial`；无法确认缺口范围时为 `unknown` |
| 超时 | `DEADLINE_EXCEEDED`，禁止返回空列表冒充 |

---

### 4.4 GetDeliveryTimeline

指定消息的**投递侧**事件；群聊按接收方拆分。

```protobuf
message GetDeliveryTimelineRequest {
  string messageId  = 1;  // 必填
  string receiverId = 2;  // 可选；过滤单个接收方
  int32  limit      = 3;
  string cursor     = 4;
}

message DeliveryEvent {
  string eventId      = 1;
  string eventType    = 2;  // delivery_* / receiver_offline / ack_*
  string messageId    = 3;
  string receiverId   = 4;
  string attemptId    = 5;
  int64  occurredAt   = 6;  // UnixNano
  string source       = 7;
  string errorCode    = 8;
  map<string, string> metadata = 9;
  // 证据强度，强制填写
  string evidence     = 10; // gateway-write | transport-ack | offline-marker | legacy
}

message GetDeliveryTimelineResponse {
  repeated DeliveryEvent events = 1;
  bool   complete   = 2;
  bool   truncated  = 3;
  string nextCursor = 4;
  string messageId  = 5;
  string coverageStatus = 6; // complete | partial | unknown
  uint64 eventsDropped = 7;
}
```

| 规则 | 说明 |
|---|---|
| 群聊 | 每个 `receiverId` 独立事件；`receiverId` 过滤必须生效 |
| `delivery_succeeded` | `evidence=gateway-write`，**禁止**表述为已送达 |
| `ack_received` | `evidence=transport-ack`，仅真实 ACK |
| 无 ACK 能力 | 不得伪造；`GetCapabilities.ack_history` 反映 `partial/unsupported` |
| 观测缺口 | `coverageStatus!=complete` 时不能根据空列表断言“没有投递事件” |

---

### 4.5 GetConnectionObservations

用户连接状态观察；**区分“当前在线”与“历史当时是否在线”**。

```protobuf
message GetConnectionObservationsRequest {
  string userId    = 1;  // 必填
  int64  at        = 2;  // 时间点查询（UnixNano）；与时间范围互斥
  int64  startTime = 3;
  int64  endTime   = 4;
  int32  limit     = 5;
  string cursor    = 6;
  bool   includeCurrent = 7; // 是否附带当前在线状态
}

message ConnectionObservation {
  string connectionId = 1;
  string instanceId   = 2;
  string state        = 3;  // "online" | "offline"
  int64  observedAt   = 4;  // UnixNano
  string reason       = 5;  // connect | disconnect | replaced | timeout | shutdown | unknown
}

message CurrentConnection {
  bool   online      = 1;
  string connectionId = 2;
  string instanceId   = 3;
  int64  observedAt   = 4;  // 当前状态观测时间
  string source       = 5;  // ws-conn-table | redis-login | unknown
}

message GetConnectionObservationsResponse {
  repeated ConnectionObservation observations = 1;  // 历史事件流
  CurrentConnection current = 2;                 // 当前状态（includeCurrent=true 时填）
  bool   complete   = 3;
  bool   truncated  = 4;
  string nextCursor = 5;
  string note       = 6;  // 如 partial-history-before-rollout
  string coverageStatus = 7; // complete | partial | unknown
  uint64 eventsDropped = 8;
}
```

| 规则 | 说明 |
|---|---|
| 历史 vs 当前 | `observations` = 历史事件；`current` = 调用时刻状态；**禁止**用当前内存表冒充历史 |
| `at` 语义 | 返回 `observedAt <= at` 的最近一次状态推演所需事件（自 startTime 起） |
| 覆盖不足 | 观测启用前的时段无事件 → `note` 标明，**禁止**编造 online/offline |
| 观测丢失 | 存在 `observation_gap` 或无法确认覆盖范围时，`coverageStatus` 必须为 `partial/unknown`，不得用 `complete=true` 表示没有连接事件 |
| 当前状态来源 | WS 连接表为准；Redis `online:users` 仅作 `source` 标注的辅助（登录态≠连接态） |
| 多实例 | 事件必须来自共享存储（message_events），单进程内存不算历史 |

`at` 与 `[startTime,endTime]` 同时给 → `INVALID_ARGUMENT`（必须二选一）。

---

### 4.6 GetCapabilities

```protobuf
message GetCapabilitiesRequest {}

message GetCapabilitiesResponse {
  string messageRecord        = 1;  // message_record
  string messageTimeline      = 2;  // message_timeline
  string deliveryEvents       = 3;  // delivery_events
  string historicalConnection = 4;  // historical_connection
  string ackHistory           = 5;  // ack_history
  string writeFailureEvents   = 6;  // write_failure_events
  string readConfirmation     = 7;  // read_confirmation
  int64  observedAt           = 8;  // UnixNano
  string observationEnabled   = 9;  // "true" | "false"（当前部署观测开关）
  string ackMode              = 10; // NoAck | OnlyAck | RigorAck
}
```

取值**只能是**：`supported` | `partial` | `unsupported`。其它值客户端应视为非法。

默认诚实声明（`observationEnabled=false` 时，与实现一致前不得上调）：

```json
{
  "message_record": "supported",
  "message_timeline": "unsupported",
  "delivery_events": "unsupported",
  "historical_connection": "unsupported",
  "ack_history": "unsupported",
  "write_failure_events": "unsupported",
  "read_confirmation": "unsupported"
}
```

| 条件 | 可上调为 supported 的项 |
|---|---|
| 观测启用、事件持久化成功，且覆盖状态可证明完整 | `message_timeline`、`delivery_events`、`write_failure_events` |
| 观测启用并稳定写入 online/offline ≥1 个完整周期 | `historical_connection` |
| `AckMode=RigorAck` 且已有真实 ACK 事件可查 | `ack_history` |
| `read_confirmed` 稳定落库 | `read_confirmation` |

若观测启用但只具备部分事件或发生丢失，返回 `partial`；关闭观测时必须返回 `unsupported`。

**禁止**为了让接口完整而虚构能力。`GetCapabilities` 必须反映**本实例实际配置**，因此允许不同环境返回不同值。

---

## 5. ServiceContext 依赖

```go
type ServiceContext struct {
    Config config.Config

    ChatLogModel    immodels.ChatLogModel        // 只读使用
    EventModel      operationsmodels.EventModel  // message_events 只读
    UserRpc         userclient.User              // FindUser / GetUserInfo
    SocialRpc       socialclient.Social          // Groupusers（派生已读名单）
}
```

读 ChatLog / Event 时的错误映射：

| 底层错误 | gRPC |
|---|---|
| context deadline | `DEADLINE_EXCEEDED` |
| mongo: no documents | `found=false`（不是错误） |
| mongo: connection refused / timeout | `UNAVAILABLE` 或 `DEADLINE_EXCEEDED`（按 errors.Is 区分） |
| 参数校验失败 | `INVALID_ARGUMENT` |
| 其它 | `INTERNAL` |

---

## 6. 配置示例

```yaml
Name: operations.rpc
ListenOn: 0.0.0.0:9100
Etcd:
  Hosts:
    - etcd:2379
  Key: operations.rpc

Mongo:
  Url: mongodb://mongo:27017
  Db: im

Telemetry:
  Endpoint: jaeger:4317

# 服务间认证（与既有机制对齐）
ServiceAuth:
  Enable: true
  # metadata key: x-im-service-token
```

---

## 7. 测试契约（与 compatibility 文档对齐）

每个 RPC 至少覆盖：

1. 正常返回
2. 空结果（OK + 空/`found=false`）
3. 非法参数（`INVALID_ARGUMENT`）
4. 超时（`DEADLINE_EXCEEDED`，**断言不是 found=false**）
5. 权限错误（`PERMISSION_DENIED`）
6. 能力不支持（`UNIMPLEMENTED`，若适用）
7. 超过最大 limit（`INVALID_ARGUMENT`）
8. 时间范围非法（`INVALID_ARGUMENT`）
9. 脱敏断言（响应中无 password/token/phone/msgContent/bitmap）
10. 群聊多接收方独立投递记录（GetDeliveryTimeline）
11. 观测缺口（`observation_gap`）不会返回 `complete=true`，空结果不被误判为“没有事件”
12. 缺少、过期或伪造请求体身份字段时，interceptor 返回 `PERMISSION_DENIED`

---

## 8. 实现顺序（闸门）

| 闸门 | 内容 | 退出标准 |
|---|---|---|
| A | proto + 事件模型 + messageId 贯通 | 旧测试全绿；新消息 ID 稳定 |
| B | ObservationSink + 五类钩子（默认 Noop） | Enabled=false 时行为与现网一致 |
| C | 连接事件 + 可选 ACK 观测 | 默认不改变 ACK；开启后有真实事件 |
| D | OperationsQuery 六 RPC + 契约测试 | §7 全覆盖 |
| E | 集成测试 + 三份文档核对 + 全量门禁 | `go test ./...` / `gofmt` / `go vet` 通过 |

详见 [compatibility-and-rollout.md](./compatibility-and-rollout.md)。
