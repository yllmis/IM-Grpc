import axios from 'axios'
import { useSessionStore } from '../store/session'

// 开发环境走 Vite 代理，生产环境走直连
const baseURL = import.meta.env.DEV ? '' : 'http://127.0.0.1:8888'

const http = axios.create({
  baseURL,
  timeout: 10000
})

// 根据接口路径选择正确的代理前缀
function getProxyBase(config) {
  if (!import.meta.env.DEV) return ''
  const url = config.url || ''
  if (url.startsWith('/v1/user')) return '/api/user'
  if (url.startsWith('/v1/social')) return '/api/social'
  if (url.startsWith('/v1/im')) return '/api/im'
  return ''
}

http.interceptors.request.use((config) => {
  const session = useSessionStore()
  if (session.token) {
    config.headers.Authorization = `Bearer ${session.token}`
  }
  // 开发环境添加代理前缀
  if (import.meta.env.DEV) {
    config.url = getProxyBase(config) + config.url
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0 && res.code !== 200) {
      return Promise.reject(new Error(res.msg || '请求失败'))
    }
    return res.data !== undefined ? res.data : res
  },
  (error) => {
    if (error.response?.status === 401) {
      const session = useSessionStore()
      session.clearSession()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export { http }
