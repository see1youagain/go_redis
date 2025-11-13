package database

import (
	"go_redis/interface/resp"
	"go_redis/lib/utils"
	"go_redis/lib/wildcard"
	"go_redis/resp/reply"
)

// 处理键相关的命令
// DEL EXISTS KEYS FLUSH TYPE RENAME RENAMENX

// DEl
func execDel(db *DB, args [][]byte) resp.Reply {
	keys := make([]string, len(args))
	for i, arg := range args {
		keys[i] = string(arg)
	}
	deleted := db.Removes(keys...)
	if deleted > 0 {
		db.addAof(utils.ToCmdLine3("del", args...))
	}
	return reply.MakeIntReply(int64(deleted))
}

// EXISTS
func execExists(db *DB, args [][]byte) resp.Reply {
	result := int64(0)
	keys := make([]string, len(args))
	for i, arg := range args {
		keys[i] = string(arg)
	}
	for _, key := range keys {
		_, exists := db.GetEntity(key)
		if exists {
			result++
		}
	}
	return reply.MakeIntReply(result)
}

// FLUSHDB
func execFlushDB(db *DB, args [][]byte) resp.Reply {
	db.Flush()
	db.addAof(utils.ToCmdLine3("flushdb", args...))
	return reply.MakeOkReply()
}

// TYPE，TYPE K1
func execType(db *DB, args [][]byte) resp.Reply {
	key := string(args[0])
	entity, exists := db.GetEntity(key)
	if !exists {
		return reply.MakeStatusReply("none") // none\r\n
	}
	switch entity.Data.(type) {
	case []byte:
		return reply.MakeStatusReply("string") // string\r\n
	}
	//TODO:实现其他数据结构
	return &reply.UnknowErrReply{}
}

// RENAME
func execRename(db *DB, args [][]byte) resp.Reply {
	oldKey := string(args[0])
	newKey := string(args[1])
	entity, exists := db.GetEntity(oldKey)
	if !exists {
		return reply.MakeErrReply("no such key")
	}
	db.data.Put(newKey, entity)
	db.data.Remove(oldKey)
	db.addAof(utils.ToCmdLine3("rename", args...))
	return reply.MakeOkReply()
}

// RENAMENX，如果newkey不存在才改名成功
func execRenamenx(db *DB, args [][]byte) resp.Reply {
	oldKey := string(args[0])
	newKey := string(args[1])
	_, exists1 := db.GetEntity(newKey)
	if exists1 {
		return reply.MakeIntReply(0) // 失败
	}
	entity, exists2 := db.GetEntity(oldKey)
	if !exists2 {
		return reply.MakeErrReply("no such key")
	}
	db.data.Put(newKey, entity)
	db.data.Remove(oldKey)
	db.addAof(utils.ToCmdLine3("renamenx", args...))
	return reply.MakeIntReply(1) // 成功
}

// KEYS，KEYS *
func execKeys(db *DB, args [][]byte) resp.Reply {
	pattern := wildcard.CompilePattern(string(args[0]))
	result := make([][]byte, 0)
	db.data.ForEach(func(key string, value interface{}) bool {
		if pattern.IsMatch(key) {
			result = append(result, []byte(key))
		}
		return true
	})
	return reply.MakeMultiBulkReply(result)
}

func init() {
	RegisterCommand("del", execDel, -2) // 一个参数是命令，一个参数是键名
	RegisterCommand("exists", execExists, -2)
	RegisterCommand("flushdb", execFlushDB, -1) // 变长，后面的参数直接丢弃
	RegisterCommand("type", execType, 2)
	RegisterCommand("rename", execRename, 3)
	RegisterCommand("renamenx", execRenamenx, 3)
	RegisterCommand("keys", execKeys, 2)
}
