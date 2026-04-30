# IM-Grpc

全栈即时通讯（IM）系统，基于 go-zero 微服务框架构建，采用 gRPC 进行服务间通信。支持单聊、群聊、好友管理、群组管理、消息已读回执等功能。

## 架构概览

系统包含 **8 个微服务**，按领域分为 User、Social、IM、Task 四个模块：

| 服务 | 类型 | 职责 |
|------|------|------|
| user-rpc / user-api | gRPC + HTTP | 用户注册、登录、资料管理 |
| social-rpc / social-api | gRPC + HTTP | 好友关系、群组管理 |
| im-rpc / im-api | gRPC + HTTP | 会话、聊天记录查询 |
| im-ws | WebSocket | 实时消息收发网关 |
| task-mq | Kafka Consumer | 后台消息持久化与转发 |

### 目录结构

```
IM-Grpc/
  apps/                  # 所有微服务应用
    user/                # 用户服务（认证、注册、资料）
      api/               # HTTP REST API 网关
      rpc/               # gRPC 服务
      models/            # MySQL 用户模型（go-zero 缓存）
    social/              # 社交服务（好友、群组）
      api/               # HTTP REST API 网关
      rpc/               # gRPC 服务
      socialmodels/      # MySQL 模型（好友、群组、请求）
    im/                  # 即时通讯服务
      api/               # HTTP REST API（会话、聊天记录）
      rpc/               # gRPC 服务
      ws/                # WebSocket 网关（实时消息）
      immodels/          # MongoDB 模型（聊天记录、会话）
    task/                # 后台任务/消息消费者
      mq/               # Kafka 消费服务
  pkg/                   # 共享工具包
  frontend/              # Vue 3 前端
  deploy/                # Dockerfile、Makefile、SQL、部署脚本
  components/            # Apisix、Sail 配置文件
  bin/                   # 预编译二进制文件
```

## 核心消息流程

### 单聊/群聊消息流

```
客户端 → WebSocket(im-ws) → Kafka(msgChatTransfer) → task-mq(消费+持久化到MongoDB) → im-ws推送 → 接收方
```

1. 客户端通过 WebSocket 连接 `im-ws`，使用 JWT 认证
2. 客户端发送 `conversation.chat` 消息
3. `im-ws` 将消息发布到 Kafka topic `msgChatTransfer`
4. `task-mq` 消费消息，持久化到 MongoDB（chat_log 集合），更新会话
5. `im-ws` 将消息实时推送给接收方（单聊推送给一人，群聊并发推送给所有成员）

### 已读回执流程

```
客户端 → WebSocket(conversation.markRead) → Kafka(msgReadTransfer) → task-mq(更新bitmap) → im-ws推送 → 发送方
```

## 技术栈

| 类别 | 技术 |
|------|------|
| 微服务框架 | go-zero v1.10.0 (zeromicro) |
| RPC | gRPC v1.79.3 + Protobuf |
| HTTP | go-zero REST Server |
| WebSocket | gorilla/websocket（自定义服务） |
| 服务发现 | etcd v3.5.15 |
| 消息队列 | Apache Kafka (segmentio/kafka-go) |
| 关系型数据库 | MySQL 5.7 |
| 文档数据库 | MongoDB 4.2 |
| 缓存 | Redis (go-redis/v9) |
| API 网关 | Apache APISIX 3.2.0 |
| 配置管理 | Sail（远程配置 + 热更新） |
| 认证 | JWT (golang-jwt/jwt/v4) |
| 密码加密 | bcrypt |
| 唯一 ID | WUID |
| 链路追踪 | OpenTelemetry + Zipkin |
| 前端 | Vue 3 + Vite + Pinia + Vue Router + Axios |

## 数据存储

### MySQL（用户与社交数据）

- `users` — 用户账号（头像、昵称、手机号、密码、性别、状态）
- `friends` — 好友关系
- `friend_requests` — 好友请求流程
- `groups` — 群组元数据（名称、图标、创建者、类型、验证方式）
- `group_members` — 群成员与角色等级
- `group_requests` — 入群请求流程

### MongoDB（IM 数据 `yllmis-im`）

- `chat_log` — 所有消息（conversationId、发送者、接收者、内容、已读记录 bitmap）
- `conversation` — 每个用户的会话状态（最后一条消息、未读数、序列号）

## 共享包（pkg/）

| 包 | 用途 |
|----|------|
| `pkg/bitmap/` | Bitmap 实现，使用 BKDRHash 追踪消息已读状态 |
| `pkg/configserver/` | 远程配置加载与热更新（基于 Sail + etcd） |
| `pkg/constants/` | 共享枚举：消息类型、聊天类型（群聊/单聊）、内容类型 |
| `pkg/ctxdata/` | 上下文工具：提取用户 ID、JWT 生成与验证 |
| `pkg/encrypt/` | 密码加密（bcrypt） |
| `pkg/interceptor/` | gRPC 拦截器：登录认证、幂等性校验 |
| `pkg/job/` | 可配置重试逻辑 |
| `pkg/middleware/` | HTTP 幂等性中间件 |
| `pkg/resultx/` | 统一 HTTP 响应格式 |
| `pkg/wuid/` | 唯一 ID 生成与会话 ID 组合函数 |
| `pkg/xerr/` | 自定义错误码与错误类型 |

## WebSocket 协议

自定义 WebSocket 服务（`apps/im/ws/websocket/`）特性：

- **JSON 消息帧**，基于 method 路由（类似 RPC-over-WebSocket）
- **JWT 认证**，从 HTTP 升级请求中解析 token
- **三级 ACK 模式**：
  - `NoAck` — 不需要确认
  - `OnlyAck` — 发送确认，不重传
  - `RigorAck` — 确认 + 线性退避重传
- **连接管理**：双向映射（用户↔连接），支持新设备登录时强制重连
- **注册路由**：`user.online`、`conversation.chat`、`conversation.markRead`、`push`

## 亮点特性

- Bitmap 实现消息已读状态追踪（紧凑二进制表示）
- 配置热更新 + 优雅重启（进程内 gRPC/HTTP 服务重启）
- 幂等性拦截器（gRPC + HTTP 双层防护）
- 自定义业务重试机制
- WebSocket 三级 ACK 模式，保证消息可靠送达
