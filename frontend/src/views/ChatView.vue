<template>
  <section class="chat-layout">
    <aside class="conversation-panel">
      <h3>会话列表</h3>
      <p>TODO: 你来实现会话拉取与渲染</p>
    </aside>

    <article class="message-panel">
      <header>
        <h3>消息区</h3>
      </header>
      <div class="message-list">
        <p>TODO: 你来实现消息列表、发送框、WebSocket 收发</p>
      </div>
    </article>
  </section>
</template>

<style scoped>
.chat-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 16px;
  min-height: 68vh;
}

.conversation-panel,
.message-panel {
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.message-list {
  margin-top: 12px;
  min-height: 52vh;
  padding: 10px;
  border-radius: 8px;
  background: #f8fafc;
}

@media (max-width: 960px) {
  .chat-layout {
    grid-template-columns: 1fr;
  }
}
</style>
