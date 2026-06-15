import { http } from './http'

export function login(payload) {
  return http.post('/v1/user/login', payload)
}

export function register(payload) {
  return http.post('/v1/user/register', payload)
}

export function getUserInfo() {
  return http.get('/v1/user/getUserInfo')
}
