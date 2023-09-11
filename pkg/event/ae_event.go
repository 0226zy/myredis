package event

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0226zy/myredis/pkg/constant"
)

type AeFileProc func(eventLoop *AeEventLoop, fd int, clientData interface{}, mask int) error
type AeTimeProc func(eventLoop *AeEventLoop, fd int64, clientData interface{}) int64
type AeEventFinalizerProc func(event *AeEventLoop, clientData interface{}) int
type AeBeForeSleepProc func(event *AeEventLoop)

// AeFileEvent 文件事件
type AeFileEvent struct {
	Mask       int
	RFileProc  AeFileProc
	WFileProc  AeFileProc
	ClientData interface{}
}

// AeTimeEvent 定时器事件
type AeTimeEvent struct {
	Id            int64
	WhenMs        int64
	ClientData    interface{}
	TimeProc      AeTimeProc
	FinalizerProc AeEventFinalizerProc
	Next          *AeTimeEvent
}

type AeFiredEvent struct {
	Fd   int
	Mask int
}

// AeEventLoop simple event loop
type AeEventLoop struct {
	maxFd           int
	epollLoop       IEpoll
	stop            int
	fileEvents      []*AeFileEvent
	firedEvents     []*AeFiredEvent
	beforeSleepProc AeBeForeSleepProc

	// 定时器事件
	timeEventHead   *AeTimeEvent
	timeEventNextId int64
}

// NewAeEventLoop create
func NewAeEventLoop() *AeEventLoop {
	ret := &AeEventLoop{
		maxFd:           -1,
		stop:            0,
		timeEventNextId: 0,
		fileEvents:      make([]*AeFileEvent, constant.AE_SETSIZE),
		firedEvents:     make([]*AeFiredEvent, constant.AE_SETSIZE),
	}

	var err error
	if ret.epollLoop, err = NewEpoll(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	for i := 0; i < constant.AE_SETSIZE; i++ {
		ret.fileEvents[i] = &AeFileEvent{Mask: constant.AE_NONE}
	}
	for i := 0; i < constant.AE_SETSIZE; i++ {
		ret.firedEvents[i] = &AeFiredEvent{Fd: -1, Mask: constant.AE_NONE}
	}
	return ret
}

// Stop 关闭事件循环
func (eventLoop *AeEventLoop) Stop() {
	eventLoop.stop = 0
}

// CreateFileEvent	create file event and add to epoll
func (eventLoop *AeEventLoop) CreateFileEvent(fd, mask int, proc AeFileProc, clientData interface{}) error {
	if fd > constant.AE_SETSIZE {
		return errors.New("invalid fd")
	}

	fe := eventLoop.fileEvents[fd]
	if fe == nil {
		fmt.Println("fe nil")
		os.Exit(1)
	}
	if err := eventLoop.epollLoop.Add(eventLoop, fd, mask); err != nil {
		return err
	}
	fmt.Printf("create fd:%d after add\n", fd)

	fe.Mask |= mask
	if (mask & constant.AE_READABLE) > 0 {
		fe.RFileProc = proc
	}

	if (mask & constant.AE_WRITABLE) > 0 {
		fe.WFileProc = proc
	}

	fe.ClientData = clientData
	if fd > eventLoop.maxFd {
		eventLoop.maxFd = fd
	}
	return nil
}

// CreateTimeEvent create time event and add to epoll
func (eventLoop *AeEventLoop) CreateTimeEvent(ms int64,
	proc AeTimeProc,
	finProc AeEventFinalizerProc,
	clientData interface{}) {

	id := eventLoop.timeEventNextId
	eventLoop.timeEventNextId += 1
	te := &AeTimeEvent{
		Id:            id,
		TimeProc:      proc,
		FinalizerProc: finProc,
		ClientData:    clientData,
		Next:          eventLoop.timeEventHead,
		WhenMs:        time.Now().UnixNano()/1e6 + ms,
	}
	eventLoop.timeEventHead = te

}

// CreateFiredEvent  create fired event and add to epoll
func (eventLoop *AeEventLoop) CreateFiredEvent() {}

// DelFileEvent del file event
func (eventLoop *AeEventLoop) DelFileEvent() {}

// DelTimeEvent del time event
func (eventLoop *AeEventLoop) DelTimeEvent(id int64) {}

// DelFiredEvent del fired event
func (eventLoop *AeEventLoop) DelFiredEvent() {}

// Main event loop
func (eventLoop *AeEventLoop) Main() {
	fmt.Println(">>> ae event loop main")
	eventLoop.stop = 0
	for eventLoop.stop != 1 {
		if eventLoop.beforeSleepProc != nil {
			eventLoop.beforeSleepProc(eventLoop)
		}
		eventLoop.processEvents(constant.AE_ALL_EVENTS)
	}
}

// Wait wait event with timeout(ms)
func (eventLoop *AeEventLoop) Wait(fd, mask int, milliseconds int64) {}

// SetBeforeSleepProc set proc before enter event loop
func (eventLoop *AeEventLoop) SetBeforeSleepProc(proc AeBeForeSleepProc) {}

// ========== interal func ==================
/* Process every pending time event, then every pending file event
*  (that may be registered by time event callbacks just processed).
*  Without special falgs the function sleeps until some file event
*  fires, or when the next time event occurrs (if any).
 */
func (eventLoop *AeEventLoop) processEvents(flags int) int {
	processed := 0
	if (flags&constant.AE_TIME_EVENTS) == 0 && (flags&constant.AE_FILE_EVENTS) == 0 {
		return 0
	}
	//fmt.Println("processEvents")

	var shortest *AeTimeEvent
	if eventLoop.maxFd != -1 || ((flags&constant.AE_TIME_EVENTS) > 0 && (flags&constant.AE_DONT_WAIT) == 0) {
		// reset timer
		shortest = eventLoop.searchNearestTimer()
	}
	timeout := int64(-1)
	if shortest != nil {
		nowMs := time.Now().UnixNano() / 1e6
		timeout = shortest.WhenMs - nowMs
		if timeout < 0 {
			timeout = -1
		}
	} else {
		timeout = 0
	}

	//fmt.Printf("processEvents timeout:%d\n", timeout)
	numEvents, err := eventLoop.epollLoop.Wait(eventLoop, timeout)
	//fmt.Printf("processEvents numEvents:%d err:%v\n", numEvents, err)

	if err != nil {
		fmt.Printf("eventLoop Wait failed:%v\n", err)
		// TODO
		return 0
	}
	for i := 0; i < numEvents; i++ {
		fileEvent := eventLoop.fileEvents[eventLoop.firedEvents[i].Fd]
		mask := eventLoop.firedEvents[i].Mask
		fd := eventLoop.firedEvents[i].Fd

		if (fileEvent.Mask & mask & constant.AE_READABLE) > 0 {
			//rfired = 1
			fileEvent.RFileProc(eventLoop, fd, fileEvent.ClientData, mask)
		}
		if (fileEvent.Mask & mask & constant.AE_WRITABLE) > 0 {
			fileEvent.WFileProc(eventLoop, fd, fileEvent.ClientData, mask)
		}
		processed += 1
	}

	// check time events
	if (flags & constant.AE_TIME_EVENTS) > 0 {
		processed += eventLoop.processTimeEvents()
	}
	return processed
}

// processTimeEvents 处理时间事件
func (eventLoop *AeEventLoop) processTimeEvents() int {
	processed := 0
	te := eventLoop.timeEventHead
	maxId := eventLoop.timeEventNextId
	for te != nil {
		if te.Id > maxId {
			te = te.Next
			continue
		}
		nowMs := time.Now().UnixNano() / 1e6
		id := te.Id
		if nowMs > te.WhenMs {
			ret := te.TimeProc(eventLoop, te.Id, te.ClientData)
			processed += 1
			if ret != int64(constant.AE_NOMORE) {
				te.WhenMs = time.Now().UnixNano()/1e6 + ret
			} else {
				eventLoop.DelTimeEvent(id)
			}

		} else {
			te = te.Next
		}
	}
	return processed
}

func (eventLoop *AeEventLoop) searchNearestTimer() *AeTimeEvent {
	te := eventLoop.timeEventHead
	var nearest *AeTimeEvent
	nearest = nil

	for te != nil {
		if nearest == nil || te.WhenMs < nearest.WhenMs {
			nearest = te
		}
		te = te.Next
	}
	return nearest
}
