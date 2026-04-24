<script setup>
import { reactive } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const form = reactive({
  phone: '',
  password: ''
})

const onSubmit = async () => {
  if (!form.phone || !form.password) {
    alert('请先输入手机号和密码')
    return
  }

  // TODO(你来实现):
  // 1. 调用后端登录接口
  // 2. 保存 token 到 store 或 localStorage
  // 3. 跳转到 /chat
  // 现在先保留最小可运行行为
  await router.push('/chat')
}
</script>

<template>
  <section class="login-card">
    <h2>登录</h2>
    <form @submit.prevent="onSubmit">
      <label>
        手机号
        <input v-model="form.phone" type="text" placeholder="请输入手机号" />
      </label>
      <label>
        密码
        <input v-model="form.password" type="password" placeholder="请输入密码" />
      </label>
      <button type="submit">进入聊天</button>
    </form>
  </section>
</template>

<style scoped>
.login-card {
  width: min(420px, 100%);
  margin: 40px auto;
  padding: 20px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.login-card h2 {
  margin: 0 0 16px;
}

form {
  display: grid;
  gap: 12px;
}

label {
  display: grid;
  gap: 6px;
  font-size: 14px;
}

input {
  height: 38px;
  padding: 0 10px;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
}

button {
  height: 40px;
  border: 0;
  border-radius: 8px;
  background: #2563eb;
  color: #fff;
  cursor: pointer;
}
</style>
