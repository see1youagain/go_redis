package database

import (
	"strings"
)

// command结构体表示一个数据库命令

var cmdTable = make(map[string]*command) // 命令表，键为命令名称，值为对应的command结构体
type command struct {
	exector ExecFunc // 执行命令的函数
	arity   int      // 命令参数个数
}

func RegisterCommand(name string, exector ExecFunc, arity int) {
	name = strings.ToLower(name)
	cmdTable[name] = &command{exector, arity}
}
