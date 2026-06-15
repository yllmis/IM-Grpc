# IM-Grpc API 接口文档

## 基础信息

- **Base URL**: `http://{host}:{port}`
- **认证方式**: JWT Bearer Token（Header: `Authorization: Bearer {token}`）
- **内容类型**: `application/json`

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

**响应**:
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

**响应**:
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

**请求头**: `Authorization: Bearer {token}`

**响应**:
```json
{
  "info": {
    "id": "1234567890",
    "mobile": "13800138000",
    "nickname": "张三",
    "sex": 1,
    "avatar": "https://example.com/avatar.jpg"
  }
}
```

---

## 2. IM 服务 (im-api, 端口 8882)

所有接口需要 JWT 认证。

### 2.1 获取聊天记录

```
GET /v1/im/chatlog?conversationId={id}&msgId={id}&count={n}
```

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversationId | string | 是 | 会话 ID |
| msgId | string | 否 | 起始消息 ID（用于分页） |
| startSendTime | int64 | 否 | 开始时间戳 |
| endSendTime | int64 | 否 | 结束时间戳 |
| count | int | 否 | 消息数量 |

**响应**:
```json
{
  "list": [
    {
      "id": "msg-001",
      "conversationId": "conv-001",
      "sendId": "user-a",
      "recvId": "user-b",
      "msgType": 1,
      "msgContent": "你好",
      "chatType": 1,
      "SendTime": 1700000000
    }
  ]
}
```

---

### 2.2 获取消息已读状态

```
POST /v1/im/chatlog/readRecords
```

**请求体**:
```json
{
  "msgId": "msg-001"
}
```

**响应**:
```json
{
  "reads": ["user-b", "user-c"],
  "unReads": ["user-d"]
}
```

---

### 2.3 获取会话列表

```
GET /v1/im/conversation
```

**响应**:
```json
{
  "conversationList": {
    "conv-001": {
      "conversationId": "conv-001",
      "ChatType": 1,
      "isShow": true,
      "seq": 10,
      "read": 5
    }
  }
}
```

---

### 2.4 更新会话

```
PUT /v1/im/conversation
```

**请求体**:
```json
{
  "conversationList": {
    "conv-001": {
      "conversationId": "conv-001",
      "ChatType": 1,
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
  "sendId": "user-a",
  "recvId": "user-b",
  "ChatType": 1
}
```

---

## 3. 社交服务 (social-api, 端口 8881)

所有接口需要 JWT 认证。

### 3.1 好友相关

#### 获取好友列表
```
GET /v1/social/friends
```

**响应**:
```json
{
  "list": [
    {
      "id": 1,
      "friend_uid": "1234567890",
      "nickname": "李四",
      "avatar": "https://example.com/avatar.jpg",
      "remark": "备注名"
    }
  ]
}
```

#### 获取好友在线状态
```
GET /v1/social/friend/online
```

**响应**:
```json
{
  "onlineList": {
    "1234567890": true,
    "0987654321": false
  }
}
```

#### 发送好友申请
```
POST /v1/social/friend/putIn
```

**请求体**:
```json
{
  "user_uid": "1234567890",
  "req_msg": "我是张三",
  "req_time": 1700000000
}
```

#### 获取好友申请列表
```
GET /v1/social/friend/putIns
```

**响应**:
```json
{
  "list": [
    {
      "id": 1,
      "user_id": "my-uid",
      "req_uid": "1234567890",
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

**响应**:
```json
{
  "list": [
    {
      "id": 1,
      "name": "技术交流群",
      "icon": "https://example.com/group.png",
      "status": 1,
      "group_type": 1,
      "is_verify": true,
      "notification": "欢迎加入",
      "notification_uid": "admin-uid"
    }
  ]
}
```

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

#### 获取群成员列表
```
GET /v1/social/group/users?group_id={id}
```

**响应**:
```json
{
  "List": [
    {
      "id": 1,
      "group_id": "group-001",
      "user_id": "user-001",
      "nickname": "张三",
      "user_avatar_url": "https://example.com/avatar.jpg",
      "role_level": 1,
      "inviter_uid": "admin-uid",
      "operator_uid": "admin-uid"
    }
  ]
}
```

#### 获取群在线成员
```
GET /v1/social/group/online?group_id={id}
```

**响应**:
```json
{
  "onlineList": {
    "user-001": true,
    "user-002": false
  }
}
```

#### 申请加群
```
POST /v1/social/group/putIn
```

**请求体**:
```json
{
  "group_id": "group-001",
  "req_msg": "我想加入",
  "req_time": 1700000000,
  "join_source": 1
}
```

#### 获取加群申请列表
```
GET /v1/social/group/putIns?group_id={id}
```

**响应**:
```json
{
  "list": [
    {
      "id": 1,
      "user_id": "user-001",
      "group_id": "group-001",
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

#### 处理加群申请
```
PUT /v1/social/group/putIn
```

**请求体**:
```json
{
  "group_req_id": 1,
  "group_id": "group-001",
  "handle_result": 1
}
```

---

## 4. WebSocket 协议 (im-ws, 端口 10090)

### 连接

```
ws://{host}:10090/ws?token={jwt-token}
```

### 消息格式

所有消息为 JSON 格式：

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

**data**:
```json
{
  "chatType": 1,
  "sendId": "sender-uid",
  "recvId": "receiver-uid",
  "conversationId": "conv-001",
  "mType": 1,
  "content": "你好"
}
```

**chatType**: 1=单聊, 2=群聊
**mType**: 1=文本, 2=图片, 3=语音

**服务器响应**:
```json
{
  "frameType": 0,
  "id": "msg-001",
  "method": "conversation.chat",
  "data": {
    "msgId": "msg-001",
    "status": "sent"
  }
}
```

#### 标记已读
```
method: "conversation.markRead"
```

**data**:
```json
{
  "conversationId": "conv-001",
  "chatType": 1,
  "recvIds": ["user-b"],
  "msgIds": ["msg-001", "msg-002"]
}
```

#### 获取在线用户
```
method: "user.online"
```

**服务器响应**:
```json
{
  "frameType": 0,
  "method": "user.online",
  "data": {
    "userIds": ["user-a", "user-b"]
  }
}
```

### 服务端推送

当收到别人发来的消息时，服务器会推送：

```json
{
  "frameType": 0,
  "id": "msg-002",
  "method": "push",
  "formId": "sender-uid",
  "data": {
    "chatType": 1,
    "sendId": "sender-uid",
    "recvId": "my-uid",
    "conversationId": "conv-001",
    "mType": 1,
    "content": "你好"
  }
}
```
