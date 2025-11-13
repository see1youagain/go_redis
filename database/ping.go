package database

import (
	"go_redis/interface/resp"
	"go_redis/resp/reply"
)

func Ping(db *DB, args [][]byte) resp.Reply {
	return reply.MakePongReply()
}

func init() {
	// 注册PING命令，在包初始化的时候会调用init函数
	// 这样可以确保PING命令在数据库启动时就可用
	RegisterCommand("ping", Ping, 1)
}
