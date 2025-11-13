package handler

import (
	"context"
	"errors"
	"go_redis/cluster"
	"go_redis/config"
	"go_redis/database"
	databaseface "go_redis/interface/database"
	"go_redis/lib/logger"
	"go_redis/lib/sync/atomic"
	"go_redis/resp/connection"
	"go_redis/resp/parser"
	"go_redis/resp/reply"
	"io"
	"net"
	"strings"
	"sync"
)

var (
	unknownErrReplyBytes = reply.MakeErrReply("ERR unknown").ToBytes()
)

type RespHandler struct {
	// 处理RESP协议的逻辑
	activeConn sync.Map
	db         databaseface.Database
	closing    atomic.Boolean
}

func (r *RespHandler) closeClient(client *connection.Connection) {
	// 关闭一个客户端连接
	_ = client.Close()
	r.db.AfterClientClose(client)
	r.activeConn.Delete(client)
}

func MakeRespHandler() *RespHandler {
	var db databaseface.Database
	if config.Properties.Self != "" && len(config.Properties.Peers) > 0 {
		db = cluster.MakeClusterDatabase()
	} else {
		db = database.NewStandaloneDatabase() // 使用EchoDatabase作为示例
	}
	return &RespHandler{
		db: db,
	}
}

func (r *RespHandler) Handler(ctx context.Context, conn net.Conn) {
	if r.closing.Get() {
		_ = conn.Close()
		return
	}
	client := connection.NewConnection(conn)
	r.activeConn.Store(client, struct{}{})
	ch := parser.ParseStream(conn) // 解析RESP协议
	for payload := range ch {
		// 如果ch不关闭，会一直循环
		if payload.Err != nil {
			if payload.Err == io.EOF ||
				errors.Is(payload.Err, io.ErrUnexpectedEOF) ||
				strings.Contains(payload.Err.Error(), "use of closed network connection") {
				// 客户端连接关闭
				logger.Info("client closed" + client.RemoteAddr().String())
				r.closeClient(client)
				return
			}
			// 协议错误
			logger.Error("protocol error:", payload.Err)
			errReply := reply.MakeErrReply(payload.Err.Error())
			err := client.Write(errReply.ToBytes())
			if err != nil {
				// 回写出错，关闭连接
				r.closeClient(client)
				logger.Error("write err:" + client.RemoteAddr().String())
				return
			}
			continue
		}
		// Exec
		if payload.Data == nil {
			// 没有数据，可能是心跳包
			continue
		}
		//fmt.Printf("payload.Data type: %T\n", payload.Data)
		switch data := payload.Data.(type) {
		case *reply.MultiBulkReply:
			exec := r.db.Exec(client, data.Args)
			if exec != nil {
				_ = client.Write(exec.ToBytes())
			} else {
				_ = client.Write(unknownErrReplyBytes)
			}
		case *reply.StatusReply:
			// 处理状态命令 (如 +PING\r\n)
			args := [][]byte{[]byte(data.Status)}
			exec := r.db.Exec(client, args)
			if exec != nil {
				_ = client.Write(exec.ToBytes())
			} else {
				_ = client.Write(unknownErrReplyBytes)
			}
		case *reply.IntReply:
			// 处理整数命令 (如 :123\r\n)
			// 整数类型通常是响应，不应该作为命令处理
			logger.Warn("received unexpected integer reply from client:", data.Code)
			errReply := reply.MakeErrReply("ERR unexpected integer from client")
			_ = client.Write(errReply.ToBytes())
		default:
			logger.Error("invalid data:", payload.Data)
			continue
		}

	}
}

func (r *RespHandler) Close() error {
	// 关闭RESP协议，全部客户端连接
	logger.Info("close resp handler")
	r.activeConn.Range(
		func(key, value any) bool {
			client := key.(*connection.Connection)
			_ = client.Close()
			return true
		},
	)
	r.db.Close()
	return nil
}
