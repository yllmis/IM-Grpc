import { http } from './http'

export function getConversations() {
  return http.get('/v1/im/conversation')
}

export function putConversations(conversationList) {
  return http.put('/v1/im/conversation', { conversationList })
}

export function getChatLog(params) {
  return http.get('/v1/im/chatlog', { params })
}

export function getReadRecords(msgId) {
  return http.post('/v1/im/chatlog/readRecords', { msgId })
}

export function setupConversation(sendId, recvId, chatType) {
  return http.post('/v1/im/setup/conversation', { sendId, recvId, ChatType: chatType })
}
