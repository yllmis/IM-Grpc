package msgtransfer

import (
	"sync"
	"time"

	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/pkg/constants"
	"github.com/zeromicro/go-zero/core/logx"
)

type groupMsgRead struct {
	mu sync.Mutex

	conversationId string

	push   *ws.Push
	pushCh chan *ws.Push // 记录已读消息的延迟处理
	count  int

	// 上次推送时间
	pushTime time.Time
	done     chan struct{}
}

func newGroupMsgRead(push *ws.Push, pushCh chan *ws.Push) *groupMsgRead {
	m := &groupMsgRead{
		conversationId: push.ConversationId,
		push:     push,
		pushCh:   pushCh,
		count:    1,
		pushTime: time.Now(),
		done:     make(chan struct{}),
	}

	go m.transfer()

	return m
}

func (m *groupMsgRead) mergePush(push *ws.Push) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.push == nil {
		m.push = push
	}
	m.count++
	for msgId, read := range push.ReadRecords {
		m.push.ReadRecords[msgId] = read
	}
}

func (m *groupMsgRead) transfer() {
	// 1.超时发送
	// 2.达到合并上限发送

	timer := time.NewTimer(GroupMsgReadRecordDelayTime / 2)
	defer timer.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-timer.C:
			m.mu.Lock()

			pushTime := m.pushTime
			val := GroupMsgReadRecordDelayTime - time.Since(pushTime)
			push := m.push
			if val > 0 && m.count < GroupMsgReadRecordDelayCount || m.push == nil { // 没有达到发送条件，继续等待
				if val > 0 {
					timer.Reset(val)
				}
				m.mu.Unlock()
				continue
			}

			m.pushTime = time.Now()
			m.push = nil // 发送后清空消息，等待新的消息合并
			m.count = 0  // 发送后重置计数器
			timer.Reset(GroupMsgReadRecordDelayTime / 2)
			m.mu.Unlock()

			// 推 送

			logx.Infof("超过合并条件，推送消息: %v", push)
			m.pushCh <- push
		default:
			m.mu.Lock()
			if m.count >= GroupMsgReadRecordDelayCount && m.push != nil {
				push := m.push
				m.push = nil // 发送后清空消息，等待新的消息合并
				m.count = 0  // 发送后重置计数器
				m.pushTime = time.Now()
				m.mu.Unlock()

				// 推送
				logx.Infof("[default]超过合并条件，推送消息: %v", push)
				m.pushCh <- push
				continue
			}

			if m.isIdle() {
				conversationId := m.conversationId
				m.mu.Unlock()
				// 使msgReadTransfer释放
				m.pushCh <- &ws.Push{
					ChatType:       constants.GroupChatType,
					ConversationId: conversationId,
				}
				continue
			}

			m.mu.Unlock()
			tempDelay := GroupMsgReadRecordDelayTime / 4
			if tempDelay > time.Second {
				tempDelay = time.Second
			}
			time.Sleep(tempDelay)
		}
	}
}

func (m *groupMsgRead) IsIdle() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isIdle()
}

func (m *groupMsgRead) isIdle() bool {
	pushTime := m.pushTime
	val := GroupMsgReadRecordDelayTime*2 - time.Since(pushTime)

	if val <= 0 && m.push == nil && m.count == 0 {
		return true
	}

	return false
}

func (m *groupMsgRead) clear() {
	select {
	case <-m.done:
	default:
		close(m.done)
	}

	m.push = nil
}
