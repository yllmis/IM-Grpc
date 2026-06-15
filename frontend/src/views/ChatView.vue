<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '../store/session'
import { ImSocket } from '../ws/imSocket'
import { getConversations, getChatLog, setupConversation } from '../api/im'
import { getFriends, getFriendOnline, getGroups, createGroup } from '../api/social'

const router = useRouter()
const session = useSessionStore()
const ws = new ImSocket()

// ChatType 常量（与后端一致）
const GROUP_CHAT = 1
const SINGLE_CHAT = 2

// 左侧导航
const activeTab = ref('chats')
const searchQuery = ref('')

// 数据
const conversations = ref([])
const activeConversation = ref(null)
const friends = ref([])
const friendOnline = ref({})
const groups = ref([])
const activeContactTab = ref('friends')

// 消息
const messages = ref([])
const inputMessage = ref('')
const messageListRef = ref(null)

// 创建群聊弹窗
const showCreateGroup = ref(false)
const newGroupName = ref('')

const myId = computed(() => session.userId)
const myNickname = computed(() => session.nickname)

// ---- 名称解析 ----
// 构建 uid -> 好友信息 的映射
const friendMap = computed(() => {
  const map = {}
  for (const f of friends.value) {
    map[f.friend_uid] = f
  }
  return map
})

// 构建 groupId -> 群信息 的映射
const groupMap = computed(() => {
  const map = {}
  for (const g of groups.value) {
    map[g.id] = g
  }
  return map
})

// 从单聊 conversationId 中提取对方的 uid
function getOtherUid(convId) {
  const parts = convId.split('_')
  if (parts.length !== 2) return convId
  return parts[0] === myId.value ? parts[1] : parts[0]
}

// 解析会话显示名称
function resolveConvName(conv) {
  if (conv.ChatType === GROUP_CHAT) {
    // 群聊：conversationId 就是群 ID
    const g = groupMap.value[conv.conversationId]
    return g?.name || `群组(${conv.conversationId})`
  }
  // 单聊：从 conversationId 提取对方 uid，查好友列表
  const otherUid = getOtherUid(conv.conversationId)
  const f = friendMap.value[otherUid]
  if (f) return f.remark || f.nickname
  // 不是好友，显示 uid 短ID
  return otherUid.length > 8 ? otherUid.slice(-8) : otherUid
}

// 过滤后的会话列表
const filteredConversations = computed(() => {
  if (!searchQuery.value) return conversations.value
  const q = searchQuery.value.toLowerCase()
  return conversations.value.filter(c =>
    resolveConvName(c).toLowerCase().includes(q)
  )
})

const filteredFriends = computed(() => {
  if (!searchQuery.value) return friends.value
  const q = searchQuery.value.toLowerCase()
  return friends.value.filter(f =>
    f.nickname?.toLowerCase().includes(q) || f.remark?.toLowerCase().includes(q)
  )
})

const filteredGroups = computed(() => {
  if (!searchQuery.value) return groups.value
  const q = searchQuery.value.toLowerCase()
  return groups.value.filter(g => g.name?.toLowerCase().includes(q))
})

onMounted(async () => {
  await loadData()
  connectWebSocket()
})

onUnmounted(() => {
  ws.close()
})

async function loadData() {
  try {
    const [convResp, friendResp, groupResp, onlineResp] = await Promise.all([
      getConversations(),
      getFriends(),
      getGroups(),
      getFriendOnline()
    ])

    friends.value = friendResp.list || []
    groups.value = groupResp.list || []
    friendOnline.value = onlineResp.onlineList || {}

    // 转换会话列表
    const convList = convResp.conversationList || {}
    conversations.value = Object.values(convList)
      .sort((a, b) => (b.seq || 0) - (a.seq || 0))
      .map(c => ({
        ...c,
        unread: Math.max(0, (c.seq || 0) - (c.read || 0)),
        lastMessage: '',
        _name: '' // 懒解析
      }))
  } catch (e) {
    console.error('加载数据失败:', e)
  }
}

function connectWebSocket() {
  ws.connect(session.token, {
    onOpen: () => console.log('WebSocket 已连接'),
    onMessage: (data) => handleWsMessage(data),
    onClose: () => console.log('WebSocket 断开'),
    onError: (e) => console.error('WebSocket 错误:', e)
  })
}

function handleWsMessage(data) {
  if (data.method === 'push' && data.data) {
    const chat = data.data
    // 添加消息到当前会话
    if (activeConversation.value?.conversationId === chat.conversationId) {
      messages.value.push({
        id: data.id,
        sendId: chat.sendId,
        content: chat.content,
        mType: chat.mType,
        time: formatTime(chat.sendTime),
        isMe: chat.sendId === myId.value
      })
      scrollToBottom()
    }
    // 更新或新建会话
    let conv = conversations.value.find(c => c.conversationId === chat.conversationId)
    if (conv) {
      conv.lastMessage = chat.content
      conv.seq = (conv.seq || 0) + 1
      if (chat.sendId !== myId.value) {
        conv.unread = (conv.unread || 0) + 1
      }
      // 移到顶部
      conversations.value.sort((a, b) => (b.seq || 0) - (a.seq || 0))
    } else {
      conversations.value.unshift({
        conversationId: chat.conversationId,
        ChatType: chat.chatType,
        lastMessage: chat.content,
        seq: 1,
        read: 0,
        unread: chat.sendId === myId.value ? 0 : 1
      })
    }
  } else if (data.method === 'conversation.chat' && data.data?.status === 'sent') {
    console.log('消息已发送:', data.data.msgId)
  }
}

async function selectConversation(conv) {
  activeConversation.value = conv
  conv.unread = 0
  messages.value = []

  try {
    const resp = await getChatLog({
      conversationId: conv.conversationId,
      count: 50
    })
    messages.value = (resp.list || []).map(m => ({
      id: m.id,
      sendId: m.sendId,
      content: m.msgContent,
      mType: m.msgType,
      time: formatTime(m.SendTime),
      isMe: m.sendId === myId.value
    })).reverse()
    scrollToBottom()
  } catch (e) {
    console.error('加载聊天记录失败:', e)
  }
}

function sendMessage() {
  const content = inputMessage.value.trim()
  if (!content || !activeConversation.value) return

  const conv = activeConversation.value
  const chatType = conv.ChatType || SINGLE_CHAT

  // 确定 recvId
  let recvId
  if (chatType === GROUP_CHAT) {
    recvId = conv.conversationId // 群聊：recvId 是群 ID
  } else {
    recvId = getOtherUid(conv.conversationId) // 单聊：recvId 是对方 uid
  }

  const chatData = {
    chatType,
    sendId: myId.value,
    recvId,
    conversationId: conv.conversationId,
    mType: 1,
    content
  }

  if (ws.sendChat(chatData)) {
    messages.value.push({
      id: Date.now().toString(),
      sendId: myId.value,
      content,
      mType: 1,
      time: new Date().toLocaleTimeString(),
      isMe: true
    })
    conv.lastMessage = content
    inputMessage.value = ''
    scrollToBottom()
  }
}

async function startChatWithFriend(friend) {
  const recvId = friend.friend_uid
  const convId = [myId.value, recvId].sort().join('_')

  let conv = conversations.value.find(c => c.conversationId === convId)
  if (!conv) {
    try {
      await setupConversation(myId.value, recvId, SINGLE_CHAT)
      conv = {
        conversationId: convId,
        ChatType: SINGLE_CHAT,
        lastMessage: '',
        seq: 0,
        read: 0,
        unread: 0
      }
      conversations.value.unshift(conv)
    } catch (e) {
      console.error('创建会话失败:', e)
      return
    }
  }

  activeTab.value = 'chats'
  selectConversation(conv)
}

async function startGroupChat(group) {
  const convId = String(group.id)
  let conv = conversations.value.find(c => c.conversationId === convId)
  if (!conv) {
    try {
      await setupConversation(myId.value, convId, GROUP_CHAT)
      conv = {
        conversationId: convId,
        ChatType: GROUP_CHAT,
        lastMessage: '',
        seq: 0,
        read: 0,
        unread: 0
      }
      conversations.value.unshift(conv)
    } catch (e) {
      console.error('创建群会话失败:', e)
      return
    }
  }

  activeTab.value = 'chats'
  selectConversation(conv)
}

async function handleCreateGroup() {
  if (!newGroupName.value.trim()) return
  try {
    await createGroup(newGroupName.value.trim())
    newGroupName.value = ''
    showCreateGroup.value = false
    await loadData()
  } catch (e) {
    console.error('创建群聊失败:', e)
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (messageListRef.value) {
      messageListRef.value.scrollTop = messageListRef.value.scrollHeight
    }
  })
}

function handleKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

function logout() {
  ws.close()
  session.clearSession()
  router.push('/login')
}

function formatTime(ts) {
  if (!ts) return ''
  const d = typeof ts === 'number' && ts < 1e12 ? new Date(ts * 1000) : new Date(ts)
  return d.toLocaleTimeString()
}

function getAvatarColor(uid) {
  const colors = ['#f38ba8', '#fab387', '#f9e2af', '#a6e3a1', '#89dceb', '#89b4fa', '#cba6f7']
  let hash = 0
  for (let i = 0; i < (uid || '').length; i++) hash = ((hash << 5) - hash + uid.charCodeAt(i)) | 0
  return colors[Math.abs(hash) % colors.length]
}

function getDisplayName(conv) {
  return resolveConvName(conv)
}

function isGroupChat(conv) {
  return conv.ChatType === GROUP_CHAT
}
</script>

<template>
  <div class="chat-layout">
    <!-- 左侧边栏 -->
    <aside class="sidebar">
      <div class="user-info">
        <div class="avatar" :style="{ background: getAvatarColor(myId) }">
          {{ myNickname?.[0] || '?' }}
        </div>
        <span class="nickname">{{ myNickname }}</span>
      </div>

      <div class="search-box">
        <input v-model="searchQuery" type="text" placeholder="搜索..." />
      </div>

      <div class="nav-tabs">
        <button :class="{ active: activeTab === 'chats' }" @click="activeTab = 'chats'">聊天</button>
        <button :class="{ active: activeTab === 'contacts' }" @click="activeTab = 'contacts'">通讯录</button>
        <button :class="{ active: activeTab === 'settings' }" @click="activeTab = 'settings'">设置</button>
      </div>

      <!-- 聊天列表 -->
      <div class="list-container" v-show="activeTab === 'chats'">
        <div
          v-for="conv in filteredConversations"
          :key="conv.conversationId"
          class="list-item"
          :class="{ active: activeConversation?.conversationId === conv.conversationId }"
          @click="selectConversation(conv)"
        >
          <div class="avatar" :style="{ background: getAvatarColor(conv.conversationId) }">
            {{ isGroupChat(conv) ? '&#128101;' : (getDisplayName(conv)[0] || '?') }}
          </div>
          <div class="item-info">
            <div class="item-top">
              <span class="item-name">{{ getDisplayName(conv) }}</span>
              <span class="chat-type-tag" v-if="isGroupChat(conv)">群</span>
            </div>
            <div class="item-bottom">
              <span class="item-preview">{{ conv.lastMessage || '暂无消息' }}</span>
              <span class="unread-badge" v-if="conv.unread > 0">{{ conv.unread > 99 ? '99+' : conv.unread }}</span>
            </div>
          </div>
        </div>
        <div v-if="filteredConversations.length === 0" class="empty-hint">暂无聊天</div>
      </div>

      <!-- 通讯录 -->
      <div class="list-container" v-show="activeTab === 'contacts'">
        <div class="contact-tabs">
          <button :class="{ active: activeContactTab === 'friends' }" @click="activeContactTab = 'friends'">好友</button>
          <button :class="{ active: activeContactTab === 'groups' }" @click="activeContactTab = 'groups'">群组</button>
        </div>

        <div v-show="activeContactTab === 'friends'">
          <div
            v-for="f in filteredFriends"
            :key="f.friend_uid"
            class="list-item"
            @click="startChatWithFriend(f)"
          >
            <div class="avatar" :style="{ background: getAvatarColor(f.friend_uid) }">
              {{ (f.nickname || '?')[0] }}
            </div>
            <div class="item-info">
              <div class="item-top">
                <span class="item-name">{{ f.remark || f.nickname }}</span>
                <span class="online-dot" :class="{ online: friendOnline[f.friend_uid] }"></span>
              </div>
              <div class="item-bottom">
                <span class="item-preview">{{ f.nickname }}</span>
              </div>
            </div>
          </div>
          <div v-if="filteredFriends.length === 0" class="empty-hint">暂无好友</div>
        </div>

        <div v-show="activeContactTab === 'groups'">
          <div class="list-item add-group" @click="showCreateGroup = true">
            <div class="avatar" style="background: #89b4fa;">+</div>
            <div class="item-info">
              <span class="item-name">创建群聊</span>
            </div>
          </div>
          <div
            v-for="g in filteredGroups"
            :key="g.id"
            class="list-item"
            @click="startGroupChat(g)"
          >
            <div class="avatar" :style="{ background: getAvatarColor(String(g.id)) }">
              {{ g.name?.[0] || '?' }}
            </div>
            <div class="item-info">
              <div class="item-top">
                <span class="item-name">{{ g.name }}</span>
              </div>
              <div class="item-bottom">
                <span class="item-preview">{{ g.notification || '暂无公告' }}</span>
              </div>
            </div>
          </div>
          <div v-if="filteredGroups.length === 0" class="empty-hint">暂无群组</div>
        </div>
      </div>

      <!-- 设置 -->
      <div class="list-container settings-panel" v-show="activeTab === 'settings'">
        <div class="setting-item">
          <span>昵称</span>
          <span class="setting-value">{{ myNickname }}</span>
        </div>
        <div class="setting-item">
          <span>ID</span>
          <span class="setting-value">{{ myId }}</span>
        </div>
        <button class="logout-btn" @click="logout">退出登录</button>
      </div>
    </aside>

    <!-- 右侧聊天区 -->
    <main class="chat-main">
      <template v-if="activeConversation">
        <header class="chat-header">
          <h3>{{ getDisplayName(activeConversation) }}</h3>
          <span class="chat-type-tag" v-if="isGroupChat(activeConversation)">群聊</span>
        </header>

        <div class="message-list" ref="messageListRef">
          <div
            v-for="msg in messages"
            :key="msg.id"
            class="message-row"
            :class="{ me: msg.isMe }"
          >
            <div class="msg-avatar" :style="{ background: getAvatarColor(msg.sendId) }">
              {{ msg.sendId?.[0] || '?' }}
            </div>
            <div class="msg-content">
              <div class="msg-bubble">{{ msg.content }}</div>
              <div class="msg-time">{{ msg.time }}</div>
            </div>
          </div>
          <div v-if="messages.length === 0" class="empty-messages">
            <div class="empty-icon">&#128172;</div>
            <p>开始聊天吧</p>
          </div>
        </div>

        <div class="input-area">
          <textarea
            v-model="inputMessage"
            placeholder="输入消息... (Enter 发送)"
            @keydown="handleKeydown"
            rows="1"
          ></textarea>
          <button class="send-btn" @click="sendMessage" :disabled="!inputMessage.trim()">
            发送
          </button>
        </div>
      </template>

      <div v-else class="no-chat-selected">
        <div class="empty-icon">&#128172;</div>
        <p>选择一个会话开始聊天</p>
        <span>或在通讯录中选择好友</span>
      </div>
    </main>

    <!-- 创建群聊弹窗 -->
    <div class="modal-overlay" v-if="showCreateGroup" @click.self="showCreateGroup = false">
      <div class="modal">
        <h3>创建群聊</h3>
        <input v-model="newGroupName" type="text" placeholder="输入群名称" @keydown.enter="handleCreateGroup" />
        <div class="modal-actions">
          <button class="cancel-btn" @click="showCreateGroup = false">取消</button>
          <button class="confirm-btn" @click="handleCreateGroup">创建</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-layout {
  display: flex;
  height: 100vh;
  background: #1e1e2e;
}

.sidebar {
  width: 300px;
  background: #181825;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #313244;
  flex-shrink: 0;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px;
  border-bottom: 1px solid #313244;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  color: #1e1e2e;
  font-weight: 600;
  flex-shrink: 0;
}

.nickname {
  font-size: 14px;
  color: #cdd6f4;
  font-weight: 500;
}

.search-box {
  padding: 10px 16px;
}

.search-box input {
  width: 100%;
  height: 34px;
  padding: 0 12px;
  background: #1e1e2e;
  border: 1px solid #313244;
  border-radius: 8px;
  color: #cdd6f4;
  font-size: 13px;
  outline: none;
}

.search-box input:focus {
  border-color: #89b4fa;
}

.search-box input::placeholder {
  color: #585b70;
}

.nav-tabs {
  display: flex;
  padding: 0 12px;
  gap: 4px;
}

.nav-tabs button {
  flex: 1;
  height: 32px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #6c7086;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.nav-tabs button.active {
  background: #313244;
  color: #cdd6f4;
}

.nav-tabs button:hover:not(.active) {
  color: #a6adc8;
}

.list-container {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.contact-tabs {
  display: flex;
  gap: 4px;
  padding: 4px 8px 8px;
}

.contact-tabs button {
  flex: 1;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #6c7086;
  font-size: 12px;
  cursor: pointer;
}

.contact-tabs button.active {
  background: #313244;
  color: #cdd6f4;
}

.list-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
}

.list-item:hover {
  background: #1e1e2e;
}

.list-item.active {
  background: #313244;
}

.item-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.item-name {
  font-size: 14px;
  color: #cdd6f4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chat-type-tag {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  background: #45475a;
  color: #a6adc8;
  flex-shrink: 0;
  margin-left: 6px;
}

.item-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.item-preview {
  font-size: 12px;
  color: #6c7086;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.unread-badge {
  background: #f38ba8;
  color: #1e1e2e;
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 10px;
  flex-shrink: 0;
}

.online-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #585b70;
  flex-shrink: 0;
}

.online-dot.online {
  background: #a6e3a1;
}

.empty-hint {
  text-align: center;
  color: #585b70;
  font-size: 13px;
  padding: 40px 0;
}

.settings-panel {
  padding: 20px 16px;
}

.setting-item {
  display: flex;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid #313244;
  font-size: 14px;
}

.setting-value {
  color: #a6adc8;
}

.logout-btn {
  width: 100%;
  height: 40px;
  margin-top: 24px;
  border: 1px solid #f38ba8;
  border-radius: 8px;
  background: transparent;
  color: #f38ba8;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.logout-btn:hover {
  background: #f38ba8;
  color: #1e1e2e;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #1e1e2e;
}

.chat-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 20px;
  border-bottom: 1px solid #313244;
  background: #181825;
}

.chat-header h3 {
  font-size: 15px;
  color: #cdd6f4;
  font-weight: 500;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message-row {
  display: flex;
  gap: 10px;
  max-width: 70%;
}

.message-row.me {
  flex-direction: row-reverse;
  align-self: flex-end;
}

.msg-avatar {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: #1e1e2e;
  font-weight: 600;
  flex-shrink: 0;
}

.msg-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.msg-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  background: #313244;
  color: #cdd6f4;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
}

.me .msg-bubble {
  background: #89b4fa;
  color: #1e1e2e;
}

.msg-time {
  font-size: 11px;
  color: #585b70;
  padding: 0 4px;
}

.me .msg-time {
  text-align: right;
}

.empty-messages {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #585b70;
}

.empty-icon {
  font-size: 48px;
  opacity: 0.3;
}

.input-area {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  padding: 12px 20px;
  border-top: 1px solid #313244;
  background: #181825;
}

.input-area textarea {
  flex: 1;
  padding: 10px 14px;
  background: #1e1e2e;
  border: 1px solid #313244;
  border-radius: 10px;
  color: #cdd6f4;
  font-size: 14px;
  resize: none;
  outline: none;
  max-height: 100px;
  font-family: inherit;
}

.input-area textarea:focus {
  border-color: #89b4fa;
}

.input-area textarea::placeholder {
  color: #585b70;
}

.send-btn {
  height: 40px;
  padding: 0 20px;
  border: none;
  border-radius: 10px;
  background: #89b4fa;
  color: #1e1e2e;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  flex-shrink: 0;
}

.send-btn:hover {
  opacity: 0.9;
}

.send-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.no-chat-selected {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #585b70;
}

.no-chat-selected p {
  font-size: 16px;
}

.no-chat-selected span {
  font-size: 13px;
  color: #45475a;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  width: 340px;
  padding: 28px;
  background: #181825;
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.modal h3 {
  margin-bottom: 18px;
  font-size: 17px;
  color: #cdd6f4;
}

.modal input {
  width: 100%;
  height: 42px;
  padding: 0 14px;
  background: #1e1e2e;
  border: 1px solid #313244;
  border-radius: 8px;
  color: #cdd6f4;
  font-size: 14px;
  outline: none;
  margin-bottom: 20px;
}

.modal input:focus {
  border-color: #89b4fa;
}

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
}

.cancel-btn, .confirm-btn {
  height: 36px;
  padding: 0 18px;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
}

.cancel-btn {
  background: #313244;
  color: #a6adc8;
}

.confirm-btn {
  background: #89b4fa;
  color: #1e1e2e;
  font-weight: 500;
}
</style>
