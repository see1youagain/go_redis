package database

import (
	"go_redis/aof"
	"go_redis/config"
	"go_redis/interface/resp"
	"go_redis/lib/logger"
	"go_redis/resp/reply"
	"strconv"
	"strings"
)

type StandaloneDatabase struct {
	dbSet      []*DB
	aofHandler *aof.AofHandler // AOF处理器
}

func NewStandaloneDatabase() *StandaloneDatabase {
	database := &StandaloneDatabase{}
	if config.Properties.Databases == 0 {
		config.Properties.Databases = 16 // 默认16个数据库
	}
	database.dbSet = make([]*DB, config.Properties.Databases)
	for i := 0; i < config.Properties.Databases; i++ {
		db := makeDB()
		db.index = i
		database.dbSet[i] = db
	}
	if config.Properties.AppendOnly == true {
		aofHandler, err := aof.NewAofHandler(database)
		if err != nil {
			logger.Error("Failed to create AOF handler:", err)
			return nil
		}
		database.aofHandler = aofHandler
		for i, db := range database.dbSet {
			dbIndex := i
			db.addAof = func(line CmdLine) {
				database.aofHandler.AddAof(dbIndex, line)
			}
		}
	}
	return database
}

// set k v
// get k
// del k1 k2 ...
// select 2
// flushdb
func (d *StandaloneDatabase) Exec(client resp.Connection, args [][]byte) resp.Reply {
	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			logger.Error("<UNK>", r)
		}
	}()
	cmd := strings.ToLower(string(args[0]))
	if cmd == "select" {
		if len(args) != 2 {
			return reply.MakeArgNumErrReply("select")
		}
		return execSelect(client, d, args[1:])
	} else {
		result := d.dbSet[client.GetDBIndex()].Exec(client, args)
		return result
	}
}

func (d StandaloneDatabase) Close() {
}

func (d StandaloneDatabase) AfterClientClose(client resp.Connection) {

}

// select 2
func execSelect(c resp.Connection, database *StandaloneDatabase, args [][]byte) resp.Reply {
	dbIndex, err := strconv.Atoi(string(args[0]))
	if err != nil {
		return reply.MakeErrReply("ERR invalid DB index")
	}
	if dbIndex < 0 || dbIndex >= len(database.dbSet) {
		return reply.MakeErrReply("ERR DB index out of range")
	}
	c.SelectDB(dbIndex)
	return reply.MakeOkReply()
}
