export class ImSocket {
  constructor() {
    this.ws = null
    this.reconnectTimer = null
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 10
    this.handlers = {}
  }

  connect(token, { onOpen, onMessage, onClose, onError }) {
    this.handlers = { onOpen, onMessage, onClose, onError }
    this.reconnectAttempts = 0
    this._doConnect(token)
  }

  _doConnect(token) {
    // 开发环境走 Vite 代理，生产环境走直连
    const isDev = import.meta.env.DEV
    const wsUrl = isDev
      ? `ws://${window.location.host}/ws?token=${token}`
      : `ws://127.0.0.1:10090/ws?token=${token}`

    this.ws = new WebSocket(wsUrl)

    this.ws.onopen = () => {
      this.reconnectAttempts = 0
      if (this.handlers.onOpen) this.handlers.onOpen()
    }

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (this.handlers.onMessage) this.handlers.onMessage(data)
      } catch {
        // ignore parse errors
      }
    }

    this.ws.onclose = () => {
      if (this.handlers.onClose) this.handlers.onClose()
      this._tryReconnect(token)
    }

    this.ws.onerror = (event) => {
      if (this.handlers.onError) this.handlers.onError(event)
    }
  }

  _tryReconnect(token) {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    this.reconnectAttempts++
    this.reconnectTimer = setTimeout(() => this._doConnect(token), delay)
  }

  send(data) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return false
    this.ws.send(JSON.stringify(data))
    return true
  }

  sendChat(chatData) {
    return this.send({
      frameType: 0,
      id: this._genId(),
      ackSeq: 0,
      method: 'conversation.chat',
      data: chatData
    })
  }

  sendMarkRead(conversationId, chatType, recvIds, msgIds) {
    return this.send({
      frameType: 0,
      id: this._genId(),
      frameType: 0x2,
      method: 'conversation.markRead',
      data: { conversationId, chatType, recvIds, msgIds }
    })
  }

  close() {
    clearTimeout(this.reconnectTimer)
    this.reconnectAttempts = this.maxReconnectAttempts
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  _genId() {
    return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
  }
}
