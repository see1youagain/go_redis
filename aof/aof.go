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
)

// aof用于记录数据库的操作日志，支持持久化和重放操作。当数据库重启时，可以通过aof文件重放操作来恢复数据。

type CmdLine = [][]byte

const aofBufferSize = 1 << 16 // AOF缓冲区大小，1MB
type payload struct {
	cmd     CmdLine // 命令字节切片
	dbIndex int
}

// 全局唯一
type AofHandler struct {
	database       database.Database
	aofFile        *os.File
	aofFileName    string
	currentDBIndex int // 当前操作的数据库索引
	aofChan        chan *payload
}

// NewAofHandler 创建一个新的AofHandler实例
func NewAofHandler(database database.Database) (*AofHandler, error) {
	handler := &AofHandler{}
	handler.aofFileName = config.Properties.AppendFilename
	handler.database = database
	handler.LoadAof() // LoadAof(handler) // 从AOF文件加载数据到数据库
	aofFile, err := os.OpenFile(handler.aofFileName, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	handler.aofFile = aofFile
	// aofChan用于接收需要写入AOF的payload
	handler.aofChan = make(chan *payload, aofBufferSize)
	go func() {
		handler.handleAof()
	}()
	return handler, nil
}

// Add payload(set k v) -> aofChan 将一个payload添加到AOF通道中，不落盘
func (handler *AofHandler) AddAof(dbIndex int, cmdLine CmdLine) {
	if config.Properties.AppendOnly && handler.aofChan != nil {
		handler.aofChan <- &payload{cmd: cmdLine, dbIndex: dbIndex}
	}
}

// handleAof payload(set k v) <- aofChan 落盘
func (handler *AofHandler) handleAof() {
	handler.currentDBIndex = 0
	for p := range handler.aofChan {
		if p.dbIndex != handler.currentDBIndex {
			data := reply.MakeMultiBulkReply(utils.ToCmdLine("select", strconv.Itoa(p.dbIndex))).ToBytes()
			_, err := handler.aofFile.Write(data)
			if err != nil {
				logger.Error(err)
				continue // 写入AOF文件失败，继续处理下一个payload，错误不重要
			}
			handler.currentDBIndex = p.dbIndex
		}
		data := reply.MakeMultiBulkReply(p.cmd).ToBytes()
		_, err := handler.aofFile.Write(data)
		if err != nil {
			logger.Error(err)
		}
	}
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
