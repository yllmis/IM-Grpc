import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import ChatView from '../views/ChatView.vue'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'login',
    component: LoginView
  },
  {
    path: '/chat',
    name: 'chat',
    component: ChatView
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
