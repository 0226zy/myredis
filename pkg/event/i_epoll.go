package event

type IEpoll interface {
	Add(eventLoop *AeEventLoop, fd, mask int) error
	Remove(fd int) error
	Wait(eventLoop *AeEventLoop, timeout int64) (int, error)
	WaitWithChan() <-chan []int
	Close() error
}
