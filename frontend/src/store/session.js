import { defineStore } from 'pinia'

export const useSessionStore = defineStore('session', {
  state: () => ({
    token: localStorage.getItem('im_token') || '',
    userId: localStorage.getItem('im_userId') || '',
    nickname: localStorage.getItem('im_nickname') || '',
    avatar: localStorage.getItem('im_avatar') || ''
  }),
  actions: {
    setSession(payload) {
      this.token = payload.token || ''
      this.userId = payload.userId || ''
      this.nickname = payload.nickname || ''
      this.avatar = payload.avatar || ''
      localStorage.setItem('im_token', this.token)
      localStorage.setItem('im_userId', this.userId)
      localStorage.setItem('im_nickname', this.nickname)
      localStorage.setItem('im_avatar', this.avatar)
    },
    clearSession() {
      this.token = ''
      this.userId = ''
      this.nickname = ''
      this.avatar = ''
      localStorage.removeItem('im_token')
      localStorage.removeItem('im_userId')
      localStorage.removeItem('im_nickname')
      localStorage.removeItem('im_avatar')
    }
  }
})
