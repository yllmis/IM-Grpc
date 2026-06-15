# IM-Grpc 前后端对接文档

## 基础信息

- **认证方式**: JWT Bearer Token（Header: `Authorization: Bearer {token}`）
- **内容类型**: `application/json`

### 通用响应格式

所有 HTTP 接口返回 go-zero 标准格式：

```json
{
  "code": 0,
  "msg": "ok",
  "data": { ... }
}
```

- `code = 0` 表示成功，非 0 为失败
- `data` 为业务数据，失败时可能为空

### 常量定义

```go
// 聊天类型
GroupChatType  = 1  // 群聊
SingleChatType = 2  // 单聊

// 消息类型
MTypeText = 0  // 文本消息

// 消息内容类型
ContentChatType = 0  // 普通消息
ContentReadType = 1  // 已读回执
```

### UID 格式

用户 ID 由 wuid 生成，格式为 64 位十六进制：`0x00000001000000001`

### conversationId 生成规则

| 类型 | 格式 | 示例 |
|------|------|------|
| 单聊 | `{较小uid}_{较大uid}`（CombineId 排序） | `0x00000001000000001_0x0000000200000001` |
| 群聊 | 群 ID 本身 | `0x00000004000000002` |

单聊的 conversationId 与发送顺序无关，始终由两个 uid 排序后拼接。

---

## 1. 用户服务 (user-api, 端口 8888)

### 1.1 用户登录

```
POST /v1/user/login
```

**请求体**:
```json
{
  "phone": "13800138000",
  "password": "123456"
}
```

**响应 data**:
```json
{
  "token": "jwt-token-string",
  "expire": 8640000
}
```

---

### 1.2 用户注册

```
POST /v1/user/register
```

**请求体**:
```json
{
  "phone": "13800138000",
  "password": "123456",
  "nickname": "张三",
  "avatar": "https://example.com/avatar.jpg",
  "sex": 1
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| sex | int32 | 0=保密, 1=男, 2=女 |

**响应 data**:
```json
{
  "token": "jwt-token-string",
  "expire": 8640000
}
```

---

### 1.3 获取用户信息

```
GET /v1/user/getUserInfo
```

**响应 data**:
```json
{
  "info": {
    "id": "0x00000001000000001",
    "mobile": "13800138000",
    "nickname": "张三",
    "sex": 1,
    "avatar": "https://example.com/avatar.jpg"
  }
}
```

---

## 2. IM 服务 (im-api, 端口 8882)

所有接口需要 JWT 认证。用户身份从 token 中获取，无需手动传入 userId。

### 2.1 获取聊天记录

```
GET /v1/im/chatlog
```

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversationId | string | 是 | 会话 ID |
| msgId | string | 否 | 起始消息 ID（分页用，从此消息之后开始加载） |
| startSendTime | int64 | 否 | 开始时间戳（秒） |
| endSendTime | int64 | 否 | 结束时间戳（秒） |
| count | int64 | 否 | 加载数量 |

**响应 data**:
```json
{
  "list": [
    {
      "id": "0x0000000a000000001",
      "conversationId": "0x00000001000000001_0x0000000200000001",
      "sendId": "0x00000001000000001",
      "recvId": "0x0000000200000001",
      "msgContent": "你好",
      "msgType": 0,
      "chatType": 2,
      "SendTime": 1700000000
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| msgType | int32 | 0=文本 |
| chatType | int32 | 1=群聊, 2=单聊 |
| SendTime | int64 | 发送时间，Unix 时间戳（秒） |

---

### 2.2 获取消息已读状态

```
POST /v1/im/chatlog/readRecords
```

**请求体**:
```json
{
  "msgId": "0x0000000a000000001"
}
```

**响应 data**:
```json
{
  "reads": ["0x0000000200000001"],
  "unReads": ["0x0000000300000001"]
}
```

---

### 2.3 获取会话列表

```
GET /v1/im/conversation
```

**响应 data**:
```json
{
  "conversationList": {
    "0x00000001000000001_0x0000000200000001": {
      "conversationId": "0x00000001000000001_0x0000000200000001",
      "ChatType": 2,
      "isShow": true,
      "seq": 10,
      "read": 5
    }
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| conversationId | string | 会话 ID（单聊为 `uid1_uid2`，群聊为群 ID） |
| ChatType | int32 | 1=群聊, 2=单聊 |
| isShow | bool | 是否显示在会话列表 |
| seq | int64 | 会话总消息数 |
| read | int32 | 已读消息数 |

**未读数计算**: `unread = seq - read`

**前端名称解析**:
- 单聊：从 conversationId 提取对方 uid，查好友列表获取 nickname/remark
- 群聊：conversationId 即群 ID，查群列表获取 name

---

### 2.4 更新会话

```
PUT /v1/im/conversation
```

**请求体**:
```json
{
  "conversationList": {
    "0x00000001000000001_0x0000000200000001": {
      "conversationId": "0x00000001000000001_0x0000000200000001",
      "ChatType": 2,
      "isShow": true,
      "seq": 10,
      "read": 10
    }
  }
}
```

---

### 2.5 创建会话

```
POST /v1/im/setup/conversation
```

**请求体**:
```json
{
  "sendId": "0x00000001000000001",
  "recvId": "0x0000000200000001",
  "ChatType": 2
}
```

| ChatType | recvId 含义 |
|----------|------------|
| 2 (单聊) | 对方用户 UID |
| 1 (群聊) | 群 ID |

---

## 3. 社交服务 (social-api, 端口 8881)

所有接口需要 JWT 认证。用户身份从 token 中获取。

### 3.1 好友相关

#### 获取好友列表
```
GET /v1/social/friends
```

**响应 data**:
```json
{
  "list": [
    {
      "id": 1,
      "friend_uid": "0x0000000200000001",
      "nickname": "李四",
      "avatar": "https://example.com/avatar.jpg",
      "remark": "老李"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| friend_uid | string | 好友的用户 ID |
| nickname | string | 好友昵称 |
| remark | string | 我对好友的备注名（优先显示） |

---

#### 获取好友在线状态
```
GET /v1/social/friend/online
```

**响应 data**:
```json
{
  "onlineList": {
    "0x0000000200000001": true,
    "0x0000000300000001": false
  }
}
```

---

#### 发送好友申请
```
POST /v1/social/friend/putIn
```

**请求体**:
```json
{
  "user_uid": "0x0000000200000001",
  "req_msg": "我是张三",
  "req_time": 1700000000
}
```

---

#### 获取好友申请列表
```
GET /v1/social/friend/putIns
```

**响应 data**:
```json
{
  "list": [
    {
      "id": 1,
      "user_id": "0x00000001000000001",
      "req_uid": "0x0000000200000001",
      "req_msg": "我是张三",
      "req_time": 1700000000,
      "handle_result": 0,
      "handle_msg": "",
      "nickname": "李四",
      "avatar": "https://example.com/avatar.jpg"
    }
  ]
}
```

| handle_result | 说明 |
|---------------|------|
| 0 | 待处理 |
| 1 | 同意 |
| 2 | 拒绝 |

---

#### 处理好友申请
```
PUT /v1/social/friend/putIn
```

**请求体**:
```json
{
  "friend_req_id": 1,
  "handle_result": 1
}
```

---

### 3.2 群组相关

#### 获取群列表
```
GET /v1/social/groups
```

**响应 data**:
```json
{
  "list": [
    {
      "id": 4,
      "name": "技术交流群",
      "icon": "https://example.com/group.png",
      "status": 1,
      "group_type": 1,
      "is_verify": true,
      "notification": "欢迎加入技术交流群",
      "notification_uid": "0x00000001000000001"
    }
  ]
}
```

---

#### 创建群
```
POST /v1/social/group
```

**请求体**:
```json
{
  "name": "技术交流群",
  "icon": "https://example.com/group.png"
}
```

---

#### 获取群成员列表
```
GET /v1/social/group/users?group_id={id}
```

**响应 data**:
```json
{
  "List": [
    {
      "id": 1,
      "group_id": "4",
      "user_id": "0x00000001000000001",
      "nickname": "张三",
      "user_avatar_url": "https://example.com/avatar.jpg",
      "role_level": 1,
      "inviter_uid": "0x00000001000000001",
      "operator_uid": "0x00000001000000001"
    }
  ]
}
```

| role_level | 说明 |
|------------|------|
| 0 | 普通成员 |
| 1 | 管理员 |
| 2 | 群主 |

---

#### 获取群在线成员
```
GET /v1/social/group/online?group_id={id}
```

**响应 data**:
```json
{
  "onlineList": {
    "0x00000001000000001": true,
    "0x0000000200000001": false
  }
}
```

---

#### 申请加群
```
POST /v1/social/group/putIn
```

**请求体**:
```json
{
  "group_id": "4",
  "req_msg": "我想加入",
  "req_time": 1700000000,
  "join_source": 1
}
```

---

#### 获取加群申请列表
```
GET /v1/social/group/putIns?group_id={id}
```

**响应 data**:
```json
{
  "list": [
    {
      "id": 1,
      "user_id": "0x0000000200000001",
      "group_id": "4",
      "req_msg": "我想加入",
      "req_time": 1700000000,
      "join_source": 1,
      "inviter_user_id": "",
      "handle_user_id": "",
      "handle_time": 0,
      "handle_result": 0
    }
  ]
}
```

---

#### 处理加群申请
```
PUT /v1/social/group/putIn
```

**请求体**:
```json
{
  "group_req_id": 1,
  "group_id": "4",
  "handle_result": 1
}
```

---

## 4. WebSocket 协议 (im-ws, 端口 10090)

### 连接

```
ws://{host}:10090/ws?token={jwt-token}
```

连接时通过 URL 参数传递 JWT token 进行认证。认证失败服务器会关闭连接。

**限制**: 每个用户只允许一个连接，新连接会挤掉旧连接。

### 消息信封格式

所有消息（包括客户端发送和服务器推送）都使用统一的 JSON 信封：

```json
{
  "frameType": 0,
  "id": "unique-message-id",
  "ackSeq": 0,
  "method": "conversation.chat",
  "formId": "sender-uid",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| frameType | int | 帧类型（见下表） |
| id | string | 消息唯一 ID（客户端生成） |
| ackSeq | int | ACK 序列号 |
| method | string | 消息方法名（路由用） |
| formId | string | 发送者 UID |
| data | object | 业务数据 |

### Frame 类型

| 值 | 名称 | 说明 |
|----|------|------|
| 0x0 | FrameData | 普通数据消息 |
| 0x1 | FramePing | 心跳保活 |
| 0x2 | FrameAck | 确认消息 |
| 0x3 | FrameNoAck | 无需确认 |
| 0x9 | FrameErr | 错误响应 |

### 方法 (method)

#### 发送消息

```
method: "conversation.chat"
```

**客户端发送 data**:
```json
{
  "chatType": 2,
  "sendId": "0x00000001000000001",
  "recvId": "0x0000000200000001",
  "conversationId": "0x00000001000000001_0x0000000200000001",
  "mType": 0,
  "content": "你好"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| chatType | int | 1=群聊, 2=单聊 |
| sendId | string | 发送者 UID（服务端会用 JWT 中的 UID 覆盖） |
| recvId | string | 接收者 UID（单聊）或 群 ID（群聊） |
| conversationId | string | 会话 ID，可为空（服务端会自动生成） |
| mType | int | 0=文本 |
| content | string | 消息内容 |

**conversationId 自动计算规则**:
- 单聊且 conversationId 为空：`CombineId(sendId, recvId)` → `{较小uid}_{较大uid}`
- 群聊且 conversationId 为空：`conversationId = recvId`（群 ID）

**服务器响应**:
```json
{
  "frameType": 0,
  "id": "client-msg-id",
  "method": "conversation.chat",
  "data": {
    "msgId": "0x0000000a000000001",
    "status": "sent"
  }
}
```

---

#### 标记已读

```
method: "conversation.markRead"
```

**客户端发送 data**:
```json
{
  "conversationId": "0x00000001000000001_0x0000000200000001",
  "chatType": 2,
  "recvId": "0x0000000200000001",
  "msgIds": ["0x0000000a000000001", "0x0000000a000000002"]
}
```

| 字段 | 说明 |
|------|------|
| chatType | 1=群聊, 2=单聊 |
| recvId | 单聊为对方 UID，群聊为群 ID |
| msgIds | 要标记为已读的消息 ID 列表 |

---

#### 获取在线用户

```
method: "user.online"
```

**服务器响应 data**:
```json
{
  "userIds": ["0x00000001000000001", "0x0000000200000001"]
}
```

---

### 服务端推送 (push)

当收到别人发来的消息时，服务器通过 WebSocket 推送：

```json
{
  "frameType": 0,
  "id": "msg-id",
  "method": "push",
  "formId": "sender-uid",
  "data": {
    "chatType": 2,
    "sendId": "0x0000000200000001",
    "recvId": "0x00000001000000001",
    "conversationId": "0x00000001000000001_0x0000000200000001",
    "mType": 0,
    "content": "你好",
    "sendTime": 1700000000,
    "msgId": "0x0000000a000000003",
    "contentType": 0
}
```

| 字段 | 说明 |
|------|------|
| sendTime | 发送时间，纳秒级时间戳 |
| contentType | 0=普通消息, 1=已读回执 |
