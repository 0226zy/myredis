package server

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/0226zy/myredis/pkg/event"
)

// tcpServer tcp server
type tcpServer struct {
	ip          string
	port        int
	listener    net.Listener
	StatNumConn int
	connHandler connectionHandler
}

type connectionHandler func(conn net.Conn) error

func newTcpServer(ip string, port int, handler connectionHandler) *tcpServer {
	return &tcpServer{ip: ip, port: port, connHandler: handler}
}

// init
func (svr *tcpServer) init() error {
	fmt.Println("tcp server init")
	listener, err := net.Listen("tcp", svr.ip+":"+strconv.Itoa(svr.port))
	if err != nil {
		fmt.Printf("Listen failed:%v\n", err)
		return err
	}
	fmt.Printf("listen:%s\n", svr.ip+strconv.Itoa(svr.port))
	svr.listener = listener
	return nil
}

func (svr *tcpServer) fd() int {
	file, err := svr.listener.(*net.TCPListener).File()
	if err != nil {
		return -1
	}
	return int(file.Fd())
}

// OnAccept
func (svr *tcpServer) onAccept(eventLoop *event.AeEventLoop, fd int, clientData interface{}, mask int) error {

	var conn net.Conn
	var err error
	for {
		conn, err = svr.listener.Accept()
		if err != nil {
			return err
		}
		break
	}

	if svr.connHandler == nil {
		return errors.New("connHandler is nil")
	}

	if err := svr.connHandler(conn); err != nil {
		svr.StatNumConn++
	}
	return nil
}
