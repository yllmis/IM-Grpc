<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '../store/session'
import { login, register, getUserInfo } from '../api/auth'

const router = useRouter()
const session = useSessionStore()

const isRegister = ref(false)
const loading = ref(false)
const error = ref('')

const form = reactive({
  phone: '',
  password: '',
  nickname: '',
  avatar: '',
  sex: 1
})

async function handleSubmit() {
  error.value = ''
  if (!form.phone || !form.password) {
    error.value = '请输入手机号和密码'
    return
  }
  if (isRegister.value && !form.nickname) {
    error.value = '请输入昵称'
    return
  }

  loading.value = true
  try {
    let resp
    if (isRegister.value) {
      resp = await register({
        phone: form.phone,
        password: form.password,
        nickname: form.nickname,
        avatar: form.avatar,
        sex: form.sex
      })
    } else {
      resp = await login({ phone: form.phone, password: form.password })
    }

    session.setSession({ token: resp.token })

    // 获取用户信息
    const userResp = await getUserInfo()
    session.setSession({
      token: resp.token,
      userId: userResp.info.id,
      nickname: userResp.info.nickname,
      avatar: userResp.info.avatar
    })

    router.push('/chat')
  } catch (e) {
    error.value = e.response?.data?.msg || '操作失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <div class="logo">&#128172;</div>
        <h2>{{ isRegister ? '注册账号' : '登录' }}</h2>
        <p class="subtitle">{{ isRegister ? '创建新账号开始聊天' : '欢迎回来' }}</p>
      </div>

      <form @submit.prevent="handleSubmit" class="login-form">
        <div class="input-group">
          <label>手机号</label>
          <input v-model="form.phone" type="tel" placeholder="请输入手机号" maxlength="11" />
        </div>
        <div class="input-group">
          <label>密码</label>
          <input v-model="form.password" type="password" placeholder="请输入密码" />
        </div>
        <template v-if="isRegister">
          <div class="input-group">
            <label>昵称</label>
            <input v-model="form.nickname" type="text" placeholder="请输入昵称" />
          </div>
          <div class="input-group">
            <label>头像URL</label>
            <input v-model="form.avatar" type="text" placeholder="可选" />
          </div>
          <div class="input-group">
            <label>性别</label>
            <div class="sex-select">
              <button type="button" :class="{ active: form.sex === 1 }" @click="form.sex = 1">男</button>
              <button type="button" :class="{ active: form.sex === 2 }" @click="form.sex = 2">女</button>
              <button type="button" :class="{ active: form.sex === 0 }" @click="form.sex = 0">保密</button>
            </div>
          </div>
        </template>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <button type="submit" class="submit-btn" :disabled="loading">
          {{ loading ? '处理中...' : (isRegister ? '注册' : '登录') }}
        </button>
      </form>

      <div class="login-footer">
        <button class="switch-btn" @click="isRegister = !isRegister">
          {{ isRegister ? '已有账号？去登录' : '没有账号？去注册' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #1e1e2e;
}

.login-card {
  width: 380px;
  padding: 40px 36px;
  background: #181825;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  font-size: 48px;
  margin-bottom: 12px;
}

.login-header h2 {
  font-size: 22px;
  color: #cdd6f4;
  margin-bottom: 6px;
}

.subtitle {
  font-size: 13px;
  color: #6c7086;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.input-group label {
  font-size: 13px;
  color: #a6adc8;
}

.input-group input {
  height: 42px;
  padding: 0 14px;
  background: #1e1e2e;
  border: 1px solid #313244;
  border-radius: 10px;
  color: #cdd6f4;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.input-group input:focus {
  border-color: #89b4fa;
}

.input-group input::placeholder {
  color: #585b70;
}

.sex-select {
  display: flex;
  gap: 8px;
}

.sex-select button {
  flex: 1;
  height: 38px;
  border: 1px solid #313244;
  border-radius: 8px;
  background: #1e1e2e;
  color: #a6adc8;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}

.sex-select button.active {
  background: #89b4fa;
  color: #1e1e2e;
  border-color: #89b4fa;
}

.error-msg {
  font-size: 13px;
  color: #f38ba8;
  text-align: center;
}

.submit-btn {
  height: 44px;
  border: none;
  border-radius: 10px;
  background: #89b4fa;
  color: #1e1e2e;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
}

.submit-btn:hover {
  opacity: 0.9;
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.login-footer {
  text-align: center;
  margin-top: 20px;
}

.switch-btn {
  background: none;
  border: none;
  color: #89b4fa;
  font-size: 13px;
  cursor: pointer;
}

.switch-btn:hover {
  text-decoration: underline;
}
</style>
