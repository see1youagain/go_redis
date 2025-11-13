package database

import (
	"go_redis/datastruct/dict"
	"go_redis/interface/database"
	"go_redis/interface/resp"
	"go_redis/resp/reply"
	"strings"
)

type DB struct {
	index  int
	data   dict.Dict
	addAof func(CmdLine)
}

// SET k v
type ExecFunc func(db *DB, args [][]byte) resp.Reply

type CmdLine = [][]byte

func makeDB() *DB {
	return &DB{
		index:  0,
		data:   dict.MakeSyncDict(),
		addAof: func(CmdLine) {},
	}
}

func (db *DB) Exec(c resp.Connection, line CmdLine) resp.Reply {
	// PING SET SETNX GET DEL
	cmdName := strings.ToLower(string(line[0]))
	cmd, ok := cmdTable[cmdName]
	if !ok {
		return reply.MakeErrReply("ERR unknown command '" + cmdName + "'")
	}
	if !validateArity(cmd.arity, line) {
		return reply.MakeArgNumErrReply(cmdName)
	}
	fun := cmd.exector
	return fun(db, line[1:]) // 执行命令，删除SET等指令

}

// SET k v
// EXISTS k1 k2 ... arity=-2，-2表示可以超过2个
func validateArity(arity int, cmdArgs [][]byte) bool {
	length := len(cmdArgs)
	if arity >= 0 {
		return arity == length
	}
	return length >= -arity
}

func (db *DB) GetEntity(key string) (*database.DataEntity, bool) {
	raw, exists := db.data.Get(key)
	if !exists {
		return nil, false
	}
	entity, _ := raw.(*database.DataEntity)
	return entity, true
}

func (db *DB) PutEntity(key string, entity *database.DataEntity) int {
	// 如果key已存在，更新操作，返回存入几个
	return db.data.Put(key, entity)
}

func (db *DB) PutIfExists(key string, entity *database.DataEntity) int {
	// 如果key已存在，更新操作，返回存入几个
	return db.data.PutIfExists(key, entity)
}

func (db *DB) PutIfAbsent(key string, entity *database.DataEntity) int {
	// 如果key已存在，更新操作，返回存入几个
	return db.data.PutIfAbsent(key, entity)
}

func (db *DB) Remove(key string) {
	db.data.Remove(key)
}

func (db *DB) Removes(key ...string) (deleted int) {
	deleted = 0
	for _, key := range key {
		if db.data.Remove(key) > 0 {
			deleted++
		}
	}
	return deleted
}

func (db *DB) Flush() {
	db.data.Clear()
}
