//go:build darwin
// +build darwin

package event

import (
	"errors"
	"sync"
	"syscall"

	"github.com/0226zy/myredis/pkg/constant"
)

type Kqueue struct {
	fd      int
	ts      syscall.Timespec
	mu      *sync.RWMutex
	changes []syscall.Kevent_t
}

func NewEpoll() (IEpoll, error) {
	p, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}

	return &Kqueue{
		fd: p,
		mu: &sync.RWMutex{},
		ts: syscall.NsecToTimespec(1e9),
	}, nil
}

func (e *Kqueue) Remove(fd int) error {

	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.changes) <= 1 {
		e.changes = nil
	} else {
		changes := make([]syscall.Kevent_t, 0, len(e.changes)-1)
		ident := uint64(fd)
		for _, ke := range e.changes {
			if ke.Ident != ident {
				changes = append(changes, ke)
			}
			e.changes = changes
		}
	}
	return nil
}
func (e *Kqueue) Add(eventLoop *AeEventLoop, fd, mask int) error {
	if e := syscall.SetNonblock(fd, true); e != nil {
		return errors.New("udev:unixSetNonblock failed")
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.changes = append(e.changes, syscall.Kevent_t{
		Ident:  uint64(fd),
		Flags:  syscall.EV_ADD | syscall.EV_EOF,
		Filter: syscall.EVFILT_READ,
	})
	return nil
}

func (e *Kqueue) Wait(eventLoop *AeEventLoop, timeout int64) (int, error) {
	events := make([]syscall.Kevent_t, 10)
	e.mu.RLock()
	changes := e.changes
	e.mu.RUnlock()
	ts := syscall.Timespec{
		Sec:  timeout / 1000,
		Nsec: (timeout % 1000) * 1000000,
	}

	n, err := syscall.Kevent(e.fd, changes, events, &ts)
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
		ee := events[i]
		if ee.Filter == syscall.EVFILT_READ {
			mask |= constant.AE_READABLE
		}
		if ee.Filter == syscall.EVFILT_WRITE {
			mask |= constant.AE_WRITABLE
		}
		eventLoop.firedEvents[i].Fd = int(ee.Ident)
		eventLoop.firedEvents[i].Mask = mask
	}
	return n, nil
}

func (e *Kqueue) WaitWithChan() <-chan []int {
	ch := make(chan []int, 10)
	return ch
}

func (e *Kqueue) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.changes = nil
	return syscall.Close(e.fd)
}
