package server

import (
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/0226zy/myredis/pkg/constant"
	"github.com/0226zy/myredis/pkg/event"
)

type RedisClient struct {
	conn  net.Conn
	flags int
}

func NewRedisClient(conn net.Conn) *RedisClient {
	return &RedisClient{conn: conn}
}

func (client *RedisClient) onRead(eventLoop *event.AeEventLoop, fd int, clientData interface{}, mask int) error {
	buf := make([]byte, constant.REDIS_IOBUF_LEN)

	n, err := client.conn.Read(buf)
	if err != nil {
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			n = 0
		} else if err == io.EOF {
			fmt.Printf("client close connection\n")
			// TODO freeclient
			return nil
		} else {
			fmt.Printf("Error reading from client:%v\n", err)
			return err
		}
	}

	//if (client.flags & constant.REDIS_BLOCKED) == 0 {
	if n > 0 {
		client.processInputData(buf)
	}
	//}
	return nil
}

func (client *RedisClient) processInputData(data []byte) {
	fmt.Printf("read from client:%s\n", string(data))
}

func (client *RedisClient) fd() int {
	f, err := client.conn.(*net.TCPConn).File()
	if err != nil {
		fmt.Printf("Get net.Conn File failed:%v\n", err)
		return -1
	}
	return int(f.Fd())
}
