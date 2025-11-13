package database

import (
	"go_redis/interface/database"
	"go_redis/interface/resp"
	"go_redis/lib/utils"
	"go_redis/resp/reply"
)

// GET
func execGet(db *DB, args [][]byte) resp.Reply {
	key := string(args[0])
	entity, exists := db.GetEntity(key)
	if !exists {
		return reply.MakeNullBulkReply()
	}
	val, ok := entity.Data.([]byte)
	//TODO:实现其他数据结构
	if !ok {
		return reply.MakeErrReply("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return reply.MakeBulkReply(val)
}

// SET, SET K V
func execSet(db *DB, args [][]byte) resp.Reply {
	key := string(args[0])
	value := args[1]
	entity := &database.DataEntity{
		Data: value,
	}
	db.PutEntity(key, entity)
	db.addAof(utils.ToCmdLine3("set", args...))
	return reply.MakeOkReply()
}

// SETNX
func execSetnx(db *DB, args [][]byte) resp.Reply {
	key := string(args[0])
	value := args[1]
	entity := &database.DataEntity{
		Data: value,
	}
	result := db.PutIfAbsent(key, entity)
	db.addAof(utils.ToCmdLine3("setnx", args...))
	return reply.MakeIntReply(int64(result))
}

// GETSET, GETSET K1 V2 -> GETSET V1
func execGetset(db *DB, args [][]byte) resp.Reply {
	key := string(args[0])
	value := args[1]
	newEntity := &database.DataEntity{
		Data: value,
	}
	oldEntity, exists := db.GetEntity(key)
	db.PutEntity(key, newEntity)
	db.addAof(utils.ToCmdLine3("getset", args...))
	if !exists {
		return reply.MakeNullBulkReply()
	}
	oldValue, ok := oldEntity.Data.([]byte)
	//TODO:实现其他数据结构
	if !ok {
		return reply.MakeErrReply("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return reply.MakeBulkReply(oldValue)
}

// STRLEN
func execStrlen(db *DB, args [][]byte) resp.Reply {
	key := string(args[0])
	entity, exists := db.GetEntity(key)
	if !exists {
		return reply.MakeNullBulkReply()
	}
	val, ok := entity.Data.([]byte)
	//TODO:实现其他数据结构
	if !ok {
		return reply.MakeErrReply("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return reply.MakeIntReply(int64(len(val)))
}
func init() {
	RegisterCommand("SET", execSet, 3)
	RegisterCommand("GET", execGet, 2)
	RegisterCommand("SETNX", execSetnx, 3)
	RegisterCommand("GETSET", execGetset, 3)
	RegisterCommand("STRLEN", execStrlen, 2)
}
