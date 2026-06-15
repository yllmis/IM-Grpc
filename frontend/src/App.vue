<script setup>
import { useSessionStore } from './store/session'
import { useRouter } from 'vue-router'
import { computed } from 'vue'

const session = useSessionStore()
const router = useRouter()
const isLoggedIn = computed(() => !!session.token)

function titlebarAction(action) {
  if (window.electronAPI) {
    window.electronAPI[action]()
  }
}
</script>

<template>
  <div class="app-shell">
    <!-- 自定义标题栏 (Electron 无边框窗口) -->
    <header class="titlebar" v-if="isLoggedIn">
      <div class="titlebar-drag">
        <span class="app-title">IM-Grpc</span>
      </div>
      <div class="titlebar-controls">
        <button @click="titlebarAction('minimize')" class="tb-btn" title="最小化">&#8722;</button>
        <button @click="titlebarAction('maximize')" class="tb-btn" title="最大化">&#9633;</button>
        <button @click="titlebarAction('close')" class="tb-btn tb-close" title="关闭">&#215;</button>
      </div>
    </header>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
  background: #1e1e2e;
  color: #cdd6f4;
  overflow: hidden;
  user-select: none;
}

::-webkit-scrollbar {
  width: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: #45475a;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #585b70;
}
</style>

<style scoped>
.app-shell {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #1e1e2e;
}

.titlebar {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #11111b;
  -webkit-app-region: drag;
  flex-shrink: 0;
}

.titlebar-drag {
  flex: 1;
  padding-left: 14px;
}

.app-title {
  font-size: 13px;
  color: #a6adc8;
  font-weight: 500;
}

.titlebar-controls {
  display: flex;
  -webkit-app-region: no-drag;
}

.tb-btn {
  width: 46px;
  height: 36px;
  border: none;
  background: transparent;
  color: #a6adc8;
  font-size: 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.tb-btn:hover {
  background: #313244;
}

.tb-close:hover {
  background: #e64553;
  color: white;
}

.app-main {
  flex: 1;
  overflow: hidden;
}
</style>
