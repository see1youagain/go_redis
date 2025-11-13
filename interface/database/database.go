package database

import "go_redis/interface/resp"

// 代表redis的业务核心

type CmdLine = [][]byte

type DataEntity struct {
	Data interface{}
}

type Database interface {
	Exec(client resp.Connection, args [][]byte) resp.Reply
	Close()
	AfterClientClose(client resp.Connection)
}
