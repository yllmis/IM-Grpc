<template>
  <div class="app-shell">
    <header class="app-header">
      <h1>IM 前端练习项目</h1>
      <p>目标：先跑通登录、会话列表、消息收发最小闭环</p>
    </header>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  background: #f3f5f7;
  color: #1f2937;
  font-family: "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
}

.app-header {
  padding: 20px 24px;
  background: #111827;
  color: #f9fafb;
}

.app-header h1 {
  margin: 0;
  font-size: 24px;
}

.app-header p {
  margin: 8px 0 0;
  font-size: 14px;
  color: #d1d5db;
}

.app-main {
  padding: 24px;
}
</style>
