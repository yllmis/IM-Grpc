import axios from 'axios'

export const http = axios.create({
  // TODO(你来配置): 改成你的 API 网关地址
  baseURL: 'http://127.0.0.1:8888',
  timeout: 10000
})

http.interceptors.request.use((config) => {
  // TODO(你来实现): 从 store/localStorage 读取 token 并注入
  // const token = '...'
  // if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (response) => response.data,
  (error) => {
    // TODO(你来实现): 统一错误提示、401 跳登录、重试策略
    return Promise.reject(error)
  }
)
