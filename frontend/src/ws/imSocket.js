export class ImSocket {
  constructor(url) {
    this.url = url
    this.ws = null
  }

  connect({ onOpen, onMessage, onClose, onError }) {
    this.ws = new WebSocket(this.url)

    this.ws.onopen = () => {
      if (typeof onOpen === 'function') onOpen()
    }

    this.ws.onmessage = (event) => {
      if (typeof onMessage === 'function') onMessage(event.data)
    }

    this.ws.onclose = () => {
      if (typeof onClose === 'function') onClose()
    }

    this.ws.onerror = (event) => {
      if (typeof onError === 'function') onError(event)
    }
  }

  send(payload) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket 未连接，无法发送消息')
    }

    this.ws.send(JSON.stringify(payload))
  }

  close() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }
}
