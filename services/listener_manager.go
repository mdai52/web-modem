package services

import "sync"

var (
	globalListenerOnce sync.Once
	globalListener     *ListenerManager
)

// ListenerManager 管理监听通道的注册与广播。
type ListenerManager struct {
	mu        sync.RWMutex
	listeners map[chan string]struct{}
}

// NewListenerManager 创建监听器管理器。
func NewListenerManager() *ListenerManager {
	return &ListenerManager{listeners: make(map[chan string]struct{})}
}

// GetGlobalListener 返回全局唯一的监听器管理器。
func GetGlobalListener() *ListenerManager {
	globalListenerOnce.Do(func() {
		globalListener = NewListenerManager()
	})
	return globalListener
}

// broadcast 向所有监听器发送消息（非阻塞，包内使用）。
func (lm *ListenerManager) broadcast(message string) {
	lm.mu.RLock()
	snapshot := make([]chan string, 0, len(lm.listeners))
	for ch := range lm.listeners {
		snapshot = append(snapshot, ch)
	}
	lm.mu.RUnlock()

	for _, listener := range snapshot {
		select {
		case listener <- message:
		default:
		}
	}
}

// addListener 注册一个新的监听通道（包内使用）。
func (lm *ListenerManager) addListener(ch chan string) {
	lm.mu.Lock()
	lm.listeners[ch] = struct{}{}
	lm.mu.Unlock()
}

// removeListener 注销监听通道并关闭它（包内使用）。
func (lm *ListenerManager) removeListener(ch chan string) {
	lm.mu.Lock()
	if _, ok := lm.listeners[ch]; ok {
		delete(lm.listeners, ch)
		close(ch)
	}
	lm.mu.Unlock()
}

// Subscribe 便捷订阅：返回通道与取消函数。
func (lm *ListenerManager) Subscribe(buffer int) (chan string, func()) {
	if buffer <= 0 {
		buffer = 100
	}
	ch := make(chan string, buffer)
	lm.addListener(ch)
	cancel := func() { lm.removeListener(ch) }
	return ch, cancel
}
