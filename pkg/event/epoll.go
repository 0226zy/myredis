//go:build linux
// +build linux

package event

import (
	"fmt"
	"sync"
	"syscall"

	"github.com/0226zy/myredis/pkg/constant"
)

type Epoll struct {
	fd int
	ts syscall.Timespec
	mu *sync.RWMutex
}

func NewEpoll() (IEpoll, error) {
	var err error
	efd, err = syscall.EpollCreate1(0)
	if err != nil {
		fmt.Printf("create epoll failed:\n", err)
		return nil, err
	}
	ret := &Epoll{
		fd: efd,
		mu: &sync.RWMutex{},
		ts: syscall.NsecToTimespec(1e9),
	}
	return ret
}

func (e *Epoll) Remove(fd int) error {

}

func (e *Epoll) Add(eventLoop *AeEventLoop, fd, mask int) error {

	op = syscall.EPOLL_CTL_MOD
	if eventLoop.fileEvents[fd].Mask == constant.AE_NONE {
		op = syscall.EPOLL_CTL_ADD
	}
	ev := syscall.EpollEvent{
		Fd:     int32(fd),
		Events: 0,
	}
	mask |= eventLoop.fileEvents[fd].Mask
	if (mask & constant.AE_READABLE) > 0 {
		ev.Events |= syscall.EPOLLIN
	}
	if (mask & constant.AE_WRITABLE) > 0 {
		ev.Events |= syscall.EPOLLOUT
	}
	err = syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, int(fd), &ev)
	if err != nil {
		fmt.Printf("add event to event failed:%v\n", err)
		return err
	}
}

func (e *Epoll) Wait(eventLoop *AeEventLoop, timeout int64) (int, error) {
	e.mu.RLock()
	changes := e.changes
	e.mu.RUnlock()
	ret := []int{}

	events := make([]syscall.EpollEvent, constant.AE_SETSIZE)
	n, err := syscall.EpollWait(e.fd, events, timeout)
	if err != nil {
		if err == syscall.EINTR {
			return 0, nil
		}
		return 0, err
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	for i := 0; i < n; i++ {
		mask := 0
		if (events[i].events & syscall.EPOLLIN) > 0 {
			mask |= constant.AE_READABLE
		}
		if (events[i].events & syscall.EPOLLOUT) > 0 {
			mask |= constant.AE_WRITABLE
		}
		eventLoop.fired[i].Fd = events[i].Fd
		eventLoop.fired[i].Mask = mask
	}
	return n, nil
}

func (e *Epoll) WaitWithChan() <-chan []int {
	ch := make(chan []int, 10)
	//go func() {
	//	for {
	//		fds, err := e.Wait(-1)
	//		if err != nil {
	//			continue
	//		}
	//		if len(fds) == 0 {
	//			continue
	//		}
	//		ch <- fds
	//	}
	//}//()
	return ch
}

func (e *Epoll) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.changes = nil
	return syscall.Close(e.fd)
}
