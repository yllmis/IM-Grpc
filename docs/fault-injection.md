# OperationsQuery 测试故障注入

故障注入只用于非生产环境的 Agent 联调。它通过 OperationsQuery 在匹配测试 ID 时返回固定响应来模拟查询异常，不修改 MongoDB、Kafka、Redis、WebSocket 或消息主链路。

## 安全边界

- `FaultInjection.Enabled` 默认为 `false`；关闭时所有请求走原有逻辑。
- `Mode: pro` 或 `Mode: production` 拒绝开启故障注入。
- 规则按固定 ID 精确匹配，不支持通配符、任意 SQL、Shell 或动态脚本。
- 故障响应带有 `fault-injection` 来源或标准 gRPC 错误，Agent 可以看到真实错误语义。
- 配置修改后需要重启 OperationsQuery，避免运行期间语义漂移。
- 测试完成后删除 Rules 并恢复 `Enabled: false`。

## 配置示例

在未提交的本地配置副本中增加：

```yaml
Mode: dev

FaultInjection:
  Enabled: true
  Rules:
    "665f1c0000000000000000aa": message_missing
    "665f1c0000000000000000ab": ack_timeout
    "fault-user-offline": receiver_offline
```

当前支持的场景：

| 场景 | 作用 | 预期 Agent 语义 |
| --- | --- | --- |
| `message_missing` | 查询成功但无消息记录 | `message_not_found` |
| `query_timeout` | 返回 `DEADLINE_EXCEEDED` | 查询失败，不能说消息不存在 |
| `wrong_message_id` | 返回与请求不同的消息 ID | Connector 拒绝事实并记录冲突 |
| `malformed_response` | 返回缺少有效观测时间的响应 | Schema/映射错误，不能生成事实 |
| `permission_denied` | 返回 `PERMISSION_DENIED` | 权限错误，不产生业务事实 |
| `delivery_empty` | 返回完整窗口内的空投递事件 | 根据消息事实判断无投递记录 |
| `delivery_timeout` | 投递查询返回超时 | 工具错误，不等于没有投递 |
| `receiver_offline` | 返回接收方离线观测 | `receiver_offline`（证据足够时） |
| `ack_timeout` | 返回 ACK 超时事件 | `ack_timeout`（证据足够时） |
| `unsupported_capability` | 返回 `UNIMPLEMENTED` | `insufficient_data` + 能力不支持 |

`write_failed` 没有加入这里。当前 OperationsQuery 没有明确的写入失败事件字段，直接伪造它会把“没有记录”误说成“写入失败”。等 IM 侧持久化失败事件有稳定来源后，再增加对应测试。

## 验证流程

1. 为每个场景使用隔离的测试 ID；
2. 开启一条规则并重启 OperationsQuery；
3. 通过 SSH 隧道让 Agent 连接本地转发端口；
4. 在 Agent 工作台中提问；
5. 检查 `DiagnosisResult`、Evidence、ToolError 和 Trace；
6. 关闭注入并重启服务；
7. 重新查询同一 ID，确认正常路径恢复。

Agent 侧示例：

```sh
GO_IM_OPERATIONS_GRPC_URL=127.0.0.1:19100 npm run test:go-im
```

故障注入测试只能证明错误语义和安全边界，不能替代真实 IM 的生产稳定性测试。
