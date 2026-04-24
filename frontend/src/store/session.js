import { defineStore } from 'pinia'

export const useSessionStore = defineStore('session', {
  state: () => ({
    token: '',
    userId: '',
    nickname: ''
  }),
  actions: {
    setSession(payload) {
      this.token = payload.token || ''
      this.userId = payload.userId || ''
      this.nickname = payload.nickname || ''
    },
    clearSession() {
      this.token = ''
      this.userId = ''
      this.nickname = ''
    }
  }
})
