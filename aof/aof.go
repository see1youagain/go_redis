package aof

import (
	"go_redis/config"
	"go_redis/interface/database"
	"go_redis/lib/logger"
	"go_redis/lib/utils"
	"go_redis/resp/connection"
	"go_redis/resp/parser"
	"go_redis/resp/reply"
	"io"
	"os"
	"strconv"
	"sync"
)

// aof用于记录数据库的操作日志，支持持久化和重放操作。当数据库重启时，可以通过aof文件重放操作来恢复数据。

type CmdLine = [][]byte

// 全局唯一
type AofHandler struct {
	database       database.Database
	aofFile        *os.File
	aofFileName    string
	currentDBIndex int        // 当前操作的数据库索引
	mu             sync.Mutex // 保护同步写入的互斥锁
}

// NewAofHandler 创建一个新的AofHandler实例
func NewAofHandler(database database.Database) (*AofHandler, error) {
	handler := &AofHandler{}
	handler.aofFileName = config.Properties.AppendFilename
	handler.database = database
	handler.LoadAof() // 从AOF文件加载数据到数据库
	// WAL 需要写权限，使用 O_WRONLY
	aofFile, err := os.OpenFile(handler.aofFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	handler.aofFile = aofFile
	return handler, nil
}

// AddAof WAL 方式：命令落盘（write + fsync）成功后才返回，保证日志先于内存状态持久化
func (handler *AofHandler) AddAof(dbIndex int, cmdLine CmdLine) {
	if !config.Properties.AppendOnly {
		return
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	// 若当前 DB 索引与命令所属 DB 不同，先写入 SELECT 命令
	if dbIndex != handler.currentDBIndex {
		selectCmd := utils.ToCmdLine("select", strconv.Itoa(dbIndex))
		data := reply.MakeMultiBulkReply(selectCmd).ToBytes()
		if _, err := handler.aofFile.Write(data); err != nil {
			logger.Error("AOF write select error:", err)
			return
		}
		handler.currentDBIndex = dbIndex
	}

	// 写入实际命令
	data := reply.MakeMultiBulkReply(cmdLine).ToBytes()
	if _, err := handler.aofFile.Write(data); err != nil {
		logger.Error("AOF write cmd error:", err)
		return
	}

	// fsync：保证数据真正写入磁盘，WAL 的关键
	if err := handler.aofFile.Sync(); err != nil {
		logger.Error("AOF fsync error:", err)
	}
}

// Close 关闭前执行最后一次 fsync
func (handler *AofHandler) Close() error {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if err := handler.aofFile.Sync(); err != nil {
		logger.Error("AOF final fsync error:", err)
	}
	return handler.aofFile.Close()
}

// LoadAof 从AOF文件中加载数据到数据库
func (handler *AofHandler) LoadAof() {
	file, err := os.Open(handler.aofFileName)
	if err != nil {
		logger.Error("open aof file err:", err)
		return
	}
	defer file.Close()
	ch := parser.ParseStream(file)
	fackConn := &connection.Connection{}
	for c := range ch {
		if c == nil {
			logger.Error("received nil payload")
			break
		}
		if c.Err != nil {
			if c.Err == io.EOF {
				break
			}
			logger.Error(c.Err)
			continue
		}
		if c.Data == nil {
			logger.Error("empty payload data")
			continue
		}
		bulkReply, ok := c.Data.(*reply.MultiBulkReply)
		if !ok {
			logger.Error("require multi bulk reply")
			continue
		}
		exec := handler.database.Exec(fackConn, bulkReply.Args)
		if reply.IsErrReply(exec) {
			logger.Error(exec)
		}
	}
}
