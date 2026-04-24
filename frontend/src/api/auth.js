import { http } from './http'

export function loginByPassword(payload) {
  // 示例：按你的后端实际接口路径调整
  return http.post('/v1/user/login', payload)
}
