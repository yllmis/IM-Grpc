# 兼容性与发布（Compatibility & Rollout）

> 本文档是**兼容性红线**与**发布闸门**的唯一依据。
> 任何改动若触及红线，必须先更新本文并走对应闸门，禁止“顺手改掉”。

相关文档：[architecture.md](./architecture.md) · [operations-query.md](./operations-query.md) · [message-event-contract.md](./message-event-contract.md)

---

## 1. 兼容性总原则

```
新增能力可以加，旧语义不能改，旧客户端不用升级，观测失败不拖垮业务。
```

---

## 2. 旧 RPC 语义（禁止变更）

以下接口**请求字段含义、返回字段含义、错误语义**保持不变：

| 服务 | RPC |
|---|---|
| User | `FindUser` |
| User | `GetUserInfo` |
| Im | `GetChatLog` |
| Im | `GetConversations` |

### 2.1 明确禁止

| 禁止项 | 说明 |
|---|---|
| 改字段含义 | 如 `SendTime` 改单位 |
| 删字段 / 重新解释旧字段 | 可加字段，不可改义 |
| 改 `sent` 语义 | `sent` = 消息进入 Kafka 处理链路，**不等于 delivered** |
| 改旧客户端消息协议 | WS 帧结构、方法名、既有字段 |
| 改 Kafka 业务流程 | topic、消费分组、既有 JSON 字段 |
| 改 ChatLog 查询行为 | `GetChatLog` 仍返回 msgContent（旧接口本来就有） |

### 2.2 允许的新增（向后兼容）

| 位置 | 新增 | 兼容性 |
|---|---|---|
| Kafka JSON | `messageId` / `clientMessageId` / `correlationId` 可选字段 | 旧消费者 `json.Unmarshal` 忽略未知字段 |
| WS 回包 | 在 `{msgId, status:"sent"}` 上**追加** `serverMessageId` | 旧客户端多收一个字段，忽略即可；`msgId` 保留 |
| proto | 新服务 `OperationsQuery` 独立部署 | 不碰旧 proto |
| ChatLog | 可加只读派生字段（不改 BSON 既有字段） | 读多写少，加字段需评估索引 |
| 配置 | `DeliveryObservation` 新段 | 默认关闭 |

WS 回包兼容示例：

```json
{
  "msgId": "客户端原始请求 ID",
  "serverMessageId": "服务端 24hex 消息 ID",
  "status": "sent"
}
```

**禁止**删除或改名 `msgId`。`status` 仍为 `"sent"`，禁止改成 `"delivered"`。

---

## 3. 用户行为兼容

新增观测代码**不得**导致：

| 红线 | 判定方式 |
|---|---|
| 消息发送失败 | 回归：观测开/关发送成功率一致 |
| Kafka 发送被阻塞 | 生产侧异步/旁路，禁止同步等事件写入 |
| WebSocket 发送逻辑改变 | `push.go` 业务返回值不变（离线仍视为业务成功） |
| 旧客户端必须升级 | 协议只增不改 |
| 默认 ACK 行为改变 | 默认 `AckMode: NoAck`，`WithAck` 注释状态保持 |
| 旧接口等待时间明显增加 | P99 对比；观测写入不得进入同步路径 |

**观测失败策略：**

```
失败 → 记错误日志 + 指标（events_dropped_total 等）
     → 不重试阻塞主路径（或有限异步重试）
     → 不影响消息发送
     → 查询侧返回 coverageStatus=partial/unknown，禁止把“无事件”解释为业务事件不存在
```

---

## 4. messageId 贯通改造（唯一写路径变更）

这是**侵入性最大**的改动，单独列红线。

### 4.1 目标行为

| 场景 | messageId 来源 |
|---|---|
| 新消息 | im-ws 在 `accepted` 时 `bson.NewObjectID().Hex()`，贯穿 Kafka → Mongo `_id` → 事件 → 回包 `serverMessageId` |
| 旧消息（Kafka 无 messageId） | task-mq 消费时 `bson.NewObjectID()`（**保持旧行为**），事件 `source=compat-legacy` |
| 客户端 msg.Id | 仅作 `clientMessageId` 透传，**禁止**直接当数据库 `_id` |

### 4.2 混合版本发布顺序（强制）

不能先发布生成 `serverMessageId` 的 `im-ws`，再发布尚未识别该字段的旧 `task-mq`。否则旧消费者会忽略 Kafka 中的新字段并重新生成 ObjectID，造成 `serverMessageId=A`、`chat_log._id=B` 的关联断裂。

发布必须拆成以下闸门：

```text
A1. 先部署支持可选 messageId/clientMessageId/correlationId 的 task-mq
A2. 用新旧两种 Kafka payload 做混合版本消费回归，确认有 messageId 时沿用、没有时走 compat-legacy
A3. 再部署生成 serverMessageId 的 im-ws
A4. 最后将 MessageIdentity.EmitServerMessageId 从 false 切换为 true
```

建议配置：

```yaml
MessageIdentity:
  EmitServerMessageId: false
```

开关切换前必须完成 A1/A2；回滚时先关闭该开关，再回滚 im-ws。已有新格式消息不能被旧消费者继续消费，除非旧消费者已具备兼容解析。

### 4.2 为什么用 ObjectID hex

- `ChatLogModel.FindOne(id)` 依赖 `bson.ObjectID`，新旧 ID 形态一致
- 不改 ChatLog `_id` 类型，旧数据可读
- `GetChatLog(msgId=...)` 对新旧消息都能查

### 4.3 回归点

1. 发送 → 落库 → 推送 全链路 ID 一致（新消息）
2. 旧格式 Kafka 消息仍能消费（无 messageId 字段）
3. `GetChatLog` 按 msgId 查询新旧消息均可用
4. WS 回包同时含 `msgId` 与 `serverMessageId`
5. 消费端**不再**对新消息重新生成 ID（断言事件 messageId == ChatLog._id）

---

## 5. ACK 兼容

### 5.1 双开关，禁止混用

| 配置 | 控制什么 | 默认 |
|---|---|---|
| `websocket.WithAck(...)` | WS 传输层是否要求 ACK / 是否重传 | **未启用（NoAck）**，`im.go` 中保持注释或显式 NoAck |
| `DeliveryObservation.AckMode` | 观测层记录哪些 ACK 事件 | `NoAck` |

启动校验：二者不一致 → **拒绝启动**（fail fast），禁止静默按其中一个跑。

### 5.2 硬性约束

| 规则 | 说明 |
|---|---|
| NoAck 不伪造 | 不写 `ack_received` / `ack_timeout` |
| OnlyAck 不伪造 | 服务端发送 ACK 后结束队列，不等待客户端 ACK；不写 `ack_received` / `ack_timeout` |
| RigorAck 只记真实 ACK | 客户端 ACK 帧到达才 `ack_received` |
| 超时才记 timeout | 仅 RigorAck 真实超时分支 |
| 不改重试 | `readAck` 退避、丢弃、errCount 逻辑禁止因观测改变 |
| 旧客户端无 ACK | `ack_history` 保持 `partial` 或 `unsupported` |

### 5.3 测试环境示例

```yaml
DeliveryObservation:
  Enabled: true
  PersistEvents: true
  AckMode: RigorAck
```

并在测试配置、文档、README 同步说明“测试环境开启了 RigorAck”。

---

## 6. 配置与默认值

```yaml
DeliveryObservation:
  Enabled: false        # 总开关：false → NoopObservationSink
  PersistEvents: true   # Enabled=true 时是否写 Mongo
  AckMode: NoAck        # NoAck | OnlyAck | RigorAck
  InstanceId: ""        # 空则 hostname
  BufferSize: 1024
```

| 环境 | 建议 |
|---|---|
| 生产初期 | `Enabled: false`（零行为变化上线） |
| 生产观察期 | `Enabled: true, PersistEvents: true, AckMode: NoAck` |
| 测试/联调 | `Enabled: true, AckMode: RigorAck` |
| 性能压测对照 | 一组开、一组关，对比吞吐与 P99 |

第一版不承诺热更新：`DeliveryObservation.Enabled` 和 `AckMode` 的修改都必须重启。这样可以保证 Sink 实例、事件覆盖状态和 ACK 队列的语义在一个进程生命周期内保持一致；后续若需要热切换，另立 `SwitchableObservationSink` 设计和测试。

---

## 7. 发布闸门

| 闸门 | 内容 | 退出标准（全部满足才进下一闸） |
|---|---|---|
| **A1** 消费兼容 | task-mq 支持可选 messageId 字段 | 新旧 Kafka payload 均可消费；有 ID 不重生成 |
| **A2** ID 与契约 | operations.proto、message_events 模型与索引 | A1 通过；旧测试全绿；契约测试通过 |
| **A3** 发送端 ID | im-ws 生成并回传 serverMessageId | A1/A2 通过；ID 贯通回归通过 |
| **B** 旁路钩子 | ObservationSink（Noop/Async/Mongo）+ 五类事件钩子 | `Enabled=false` 时链路行为与改前一致；钩子失败不影响发送 |
| **C** 连接与 ACK | online/offline 事件、可选 ACK 观测 | 默认无行为变化；开启后事件真实；双开关校验生效 |
| **D** 查询服务 | OperationsQuery 六 RPC + 错误语义 + 权限 | 契约测试全覆盖（含超时≠found=false）；脱敏断言通过 |
| **E** 收口 | 集成测试、三份文档核对、CI 全绿 | `go test ./...` + `gofmt` + `go vet`；旧接口回归；文档与实现一致 |

### 7.1 闸门 A 详细清单（先做）

- [ ] `apps/operations/rpc/operations.proto`（可先只含模型 + 服务骨架）
- [ ] `pkg/observation/event.go` 字段与 eventType 常量与契约一致
- [ ] `message_events` 集合 + 5 类索引（含 `eventId` unique）
- [ ] `mq.MsgChatTransfer` 增加三字段，旧消费者兼容
- [ ] im-ws 生成并回传 `serverMessageId`
- [ ] task-mq 消费端：有 messageId 用之，无则旧行为
- [ ] 单测：ID 贯通、旧格式兼容、时间单位、幂等去重
- [ ] 旧 RPC 测试与手工发送回归

---

## 8. 测试要求

### 8.1 单元测试（必须，`go test ./...` 可跑，无外部依赖）

| 主题 | 断言要点 |
|---|---|
| 事件类型映射 | 每个挂载点 → 正确 eventType |
| 事件幂等 | 同 eventId 写两次仅一条 |
| 重复事件去重 | 并发写同 eventId 不炸、不重复 |
| 消息 ID 贯通 | accepted→kafka→persisted 同一 messageId |
| 旧消息兼容 | 无 messageId 的 payload 走 compat-legacy |
| 错误码映射 | xerr / mongo → gRPC 状态表 |
| 空结果语义 | found=false ≠ 错误；超时 ≠ found=false |
| 时间转换 | UnixNano 往返一致；非法时间拒绝 |
| 数据脱敏 | 事件与查询响应无密码/Token/正文/手机号/bitmap |
| 能力声明 | 默认值与契约一致；禁止虚报 supported |
| ACK 开关 | NoAck 不产生 ack_*；不一致配置拒绝启动 |
| WS 回包兼容 | 同时含 msgId 与 serverMessageId |

### 8.2 RPC 契约测试

每个 `OperationsQuery` RPC 覆盖：

正常 · 空结果 · 非法参数 · 超时 · 权限错误 · 不支持能力 · 超过最大 limit · 时间范围非法

### 8.3 集成测试

- 构建标签：`//go:build integration`（与现有一致，CI 默认不跑）
- 使用**测试** MongoDB / Kafka / WebSocket，**禁止生产凭证**
- 链路：`WS → Kafka → MongoDB → 投递 → ACK → OperationsQuery`
- 旧接口原测试必须继续通过

### 8.4 回归对照

| 对照项 | 方法 |
|---|---|
| 发送成功率 | 观测开 vs 关 |
| 端到端延迟 P99 | 观测开 vs 关，偏差需说明 |
| 旧 WS 客户端 | 无 `serverMessageId` 也能正常收发 |
| 旧 Kafka 消息 | 可消费、可查询（legacy） |

---

## 9. 事件最终一致性（必须对外说明）

| 事实 | 说明 |
|---|---|
| Mongo 4.2 单实例（docker-compose） | **无多文档事务** |
| ChatLog 与 message_events | **不能**同事务提交 |
| Outbox | 本期不实现 |
| 保证 | **最终一致**；事件可能延迟、可能丢失 |
| 禁止表述 | “事件与消息绝对同时成功”“审计零丢失” |

查询侧必须能表达证据不足：`truncated` / `note` / `eventsAvailable` / capabilities 的 `partial`。

---

## 10. 安全与权限红线

| 红线 | 说明 |
|---|---|
| OperationsQuery 只读 | 无 ExecuteSQL / Shell / Resend / Modify / Delete / KickUser / SubmitIncident |
| 权限在服务身份 | gRPC metadata，不在请求体 |
| 不信任 actorId/tenantId/permission | 调用方提交的不可作可信身份 |
| 不虚构租户隔离 | 无租户模型就如实写“无租户隔离” |
| 脱敏 | 见事件契约 §7；查询响应同样适用 |
| 权限错误不泄露存在性 | 一律 `PERMISSION_DENIED` |

---

## 11. 部署顺序

```
1. 按 A1→A2→A3 顺序发布 task-mq、契约和 im-ws；`MessageIdentity.EmitServerMessageId=false`
2. 发布 operations-rpc（只读查询，可独立验证）
3. 测试环境重启后 `Enabled=true + RigorAck`，跑集成测试
4. 生产环境重启后 `Enabled=true + NoAck`，观察指标、coverageStatus 与事件量
5. 需要 ACK 历史时，单独评估并重启切换到 RigorAck（默认仍 NoAck）
```

回滚：`DeliveryObservation.Enabled=false` 即回到无观测行为；查询服务可继续运行（只是事件变少）。

---

## 12. 验收清单（对照任务验收标准）

| # | 标准 | 验证方式 |
|---|---|---|
| 1 | 旧 RPC 编译与原有测试通过 | `go build ./...` + `go test ./...` |
| 2 | 旧客户端行为不改变 | 回归 §8.4 |
| 3 | OperationsQuery 可独立调用 | 契约测试 |
| 4 | 消息查询区分 找到/无记录/超时/DB错误 | 错误语义测试 |
| 5 | 新消息稳定 messageId | ID 贯通测试 |
| 6 | Kafka/Mongo/投递可按 messageId 关联 | 集成测试 |
| 7 | 投递成功/失败/离线有明确事件 | 事件测试 |
| 8 | ACK 默认不改行为 | 默认配置对照 |
| 9 | 开启 ACK 后真实 ack_received/ack_timeout | 测试环境 RigorAck |
| 10 | 事件重复写入无重复 | 幂等测试 |
| 11 | 不返回密码/Token/正文/原始日志 | 脱敏测试 |
| 12 | 无任意 SQL/Shell/危险写接口 | 代码评审 + proto 审查 |
| 13 | 无真实事件时返回能力不足/证据不足 | capabilities + truncated |
| 14 | 工具超时不转换为消息不存在 | 错误语义测试 |
| 15 | go test / gofmt / go vet 通过 | CI |
| 16 | 不依赖另一 Agent 项目编译 | go.mod / import 审查 |

---

## 13. 文档同步规则

改实现必须同步文档的场景：

| 变更 | 必须更新 |
|---|---|
| eventType / 事件字段 / 语义 | message-event-contract.md + 本文 |
| RPC / 字段 / 错误语义 | operations-query.md + 本文 |
| 默认配置 / ACK / 发布策略 | 本文 + message-event-contract.md §9.3 |
| 能力真实状态变化 | operations-query.md §4.6 + 实现 |

三份文档与代码不一致时，**以“先改文档再改代码”为流程**；紧急修复后 24h 内补文档。
