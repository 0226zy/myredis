package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/0226zy/myredis/pkg/config"
	"github.com/0226zy/myredis/pkg/constant"
	"github.com/0226zy/myredis/pkg/event"
)

// RedisServer redis server
type RedisServer struct {
	conf           *config.RedisConfig
	eventLoop      *event.AeEventLoop
	ioReadyClients []*RedisClient
	tcpServer      *tcpServer
	clients        []*RedisClient
}

// NewRedisServer create with config
func NewRedisServer(redisConf *config.RedisConfig) *RedisServer {
	return &RedisServer{
		conf:           redisConf,
		eventLoop:      event.NewAeEventLoop(),
		clients:        []*RedisClient{},
		ioReadyClients: []*RedisClient{},
	}
}

func (svr *RedisServer) Init() {

	svr.tcpServer = newTcpServer(svr.conf.Bind, svr.conf.Port, svr.createClient)
	if err := svr.tcpServer.init(); err != nil {
		fmt.Printf("Opeing TCP ip:%s port:%d failed:%v\n", svr.conf.Bind, svr.conf.Port, err)
		os.Exit(1)
	}

	if err := svr.eventLoop.CreateFileEvent(svr.tcpServer.fd(), constant.AE_READABLE,
		func(eventLoop *event.AeEventLoop, fd int, clientData interface{}, mask int) error {
			return svr.tcpServer.onAccept(eventLoop, fd, clientData, mask)
		}, nil); err != nil {
		fmt.Printf("create file event failed:%v\n", err)
		os.Exit(1)
	}

	// TODO open appendonly file

}

// Serve 主循环
func (svr *RedisServer) Serve() {
	conf := svr.conf
	fmt.Printf("The Server is now ready to accept connections on %s:%d\n", conf.Bind, conf.Port)

	svr.eventLoop.SetBeforeSleepProc(func(eventLoop *event.AeEventLoop) {
		svr.beforeSleep()
	})
	// loop
	svr.eventLoop.Main()
}

// Clear 退出前清理资源
func (svr *RedisServer) Clear() {}

// ======================= internal func ===========================
// beforeSleep
/* This function gets called every time Redis is entering the
*  main loop of the event driven library, that is, before to sleep
*  for ready file descriptors.
 */
func (svr *RedisServer) beforeSleep() {
	conf := svr.conf
	if conf.VmEnabled && len(svr.ioReadyClients) > 0 {
	}
}

func (svr *RedisServer) createClient(conn net.Conn) error {

	file, err := conn.(*net.TCPConn).File()
	if err != nil {
		return err
	}
	fd := int(file.Fd())
	if err := syscall.SetNonblock(fd, true); err != nil {
		fmt.Printf("set TCP_NOBLOCK faield:%v\n", err)
		return err
	}

	if err := conn.(*net.TCPConn).SetNoDelay(true); err != nil {
		fmt.Printf("set TCP_NODELAY faield:%v\n", err)
		return err
	}

	if svr.limitClient() {
		// 达到最大链接限制
		conn.Write([]byte("-ERR max number of clients reached\r\n"))
		return errors.New("max number of clients reached")
	}

	client := NewRedisClient(conn)
	if err := svr.eventLoop.CreateFileEvent(client.fd(), constant.AE_READABLE, client.onRead, client); err != nil {
		fmt.Printf("create file event faield:%v\n", err)
		return err
	}

	svr.clients = append(svr.clients, client)
	return nil

}

func (svr *RedisServer) freeClient(client *RedisClient) {}

func (svr *RedisServer) limitClient() bool {

	if svr.conf.MaxClients > 0 && len(svr.clients) >= svr.conf.MaxClients {
		return true
	}
	return false
}
