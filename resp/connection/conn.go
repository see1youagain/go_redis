package connection

import (
	"go_redis/lib/sync/wait"
	"net"
	"sync"
	"time"
)

type Connection struct {
	// 连接的唯一描述
	conn         net.Conn
	waitingReply wait.Wait // 保证任务完整做完
	mu           sync.Mutex
	selectedDB   int // 选择的数据库
}

func (c *Connection) Write(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	c.mu.Lock() // 避免并发写
	c.waitingReply.Add(1)
	defer func() {
		c.mu.Unlock()
		c.waitingReply.Done()
	}()
	_, err := c.conn.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}
func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		conn: conn,
	}
}

func (c *Connection) GetDBIndex() int {
	return c.selectedDB
}

func (c *Connection) SelectDB(dbNum int) {
	c.selectedDB = dbNum
}
func (c *Connection) Close() error {
	c.waitingReply.WaitWithTimeout(time.Second * 10)
	_ = c.conn.Close()
	return nil
}
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
