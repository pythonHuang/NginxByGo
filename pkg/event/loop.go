package event

import (
	"sync"
	"syscall"
	"time"
)

// Event 事件
type Event struct {
	Handler       func()
	Timer         *time.Timer
	Active        bool
	ReadEnabled   bool
	WriteEnabled  bool
}

// EventLoop 事件循环
type EventLoop struct {
	fd       int
	events   []syscall.EpollEvent
	conns    map[int]*Connection
	handler  func(int) error
	mu       sync.RWMutex
	running  bool
	stop     chan struct{}
}

// Connection 连接
type Connection struct {
	FD        int
	ReadEvent *Event
	WriteEvent *Event
	Handler   interface{}
}

// NewEventLoop 创建事件循环
func NewEventLoop() (*EventLoop, error) {
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &EventLoop{
		fd:      fd,
		events:  make([]syscall.EpollEvent, 128),
		conns:   make(map[int]*Connection, 1024),
		running: false,
		stop:    make(chan struct{}),
	}, nil
}

// Add 添加连接
func (e *EventLoop) Add(conn *Connection) error {
	var ev syscall.EpollEvent
	ev.Fd = int32(conn.FD)

	flags := syscall.EPOLLIN | syscall.EPOLLET
	if conn.WriteEvent != nil && conn.WriteEvent.WriteEnabled {
		flags |= syscall.EPOLLOUT
	}

	ev.Events = uint32(flags)
	
	if err := syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, conn.FD, &ev); err != nil {
		return err
	}

	e.mu.Lock()
	e.conns[conn.FD] = conn
	e.mu.Unlock()

	return nil
}

// Mod 修改事件
func (e *EventLoop) Mod(conn *Connection) error {
	var ev syscall.EpollEvent
	ev.Fd = int32(conn.FD)

	flags := syscall.EPOLLIN | syscall.EPOLLET
	if conn.WriteEvent != nil && conn.WriteEvent.WriteEnabled {
		flags |= syscall.EPOLLOUT
	}

	ev.Events = uint32(flags)

	return syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_MOD, conn.FD, &ev)
}

// Del 删除连接
func (e *EventLoop) Del(fd int) error {
	if err := syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil); err != nil {
		return err
	}

	e.mu.Lock()
	delete(e.conns, fd)
	e.mu.Unlock()

	return nil
}

// Wait 等待事件
func (e *EventLoop) Wait(timeout time.Duration) (int, error) {
	n, err := syscall.EpollWait(e.fd, e.events, int(timeout.Milliseconds()))
	if err != nil {
		if err == syscall.EINTR {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

// Start 启动事件循环
func (e *EventLoop) Start(acceptFn func() (*Connection, error)) error {
	e.running = true

	for {
		select {
		case <-e.stop:
			e.running = false
			return nil
		default:
			n, err := e.Wait(100 * time.Millisecond())
			if err != nil {
				return err
			}

			for i := 0; i < n; i++ {
				fd := int(e.events[i].Fd)
				ev := e.events[i].Events

				// 检查是否是监听 fd
				// 这里需要检查是否有 accept 回调

				// 处理读写事件
				if ev&syscall.EPOLLIN != 0 {
					e.mu.RLock()
					conn := e.conns[fd]
					e.mu.RUnlock()

					if conn != nil && conn.ReadEvent != nil && conn.ReadEvent.Handler != nil {
						conn.ReadEvent.Handler()
					}
				}

				if ev&syscall.EPOLLOUT != 0 {
					e.mu.RLock()
					conn := e.conns[fd]
					e.mu.RUnlock()

					if conn != nil && conn.WriteEvent != nil && conn.WriteEvent.Handler != nil {
						conn.WriteEvent.Handler()
					}
				}
			}
		}
	}
}

// Stop 停止事件循环
func (e *EventLoop) Stop() error {
	if !e.running {
		return nil
	}

	e.stop <- struct{}{}
	e.running = false
	return syscall.Close(e.fd)
}

// Close 关闭
func (e *EventLoop) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for fd := range e.conns {
		syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	}

	e.conns = make(map[int]*Connection)
	return syscall.Close(e.fd)
}