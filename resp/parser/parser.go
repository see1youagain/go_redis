package parser

import (
	"bufio"
	"errors"
	"go_redis/interface/resp"
	"go_redis/lib/logger"
	"go_redis/resp/reply"
	"io"
	"runtime/debug"
	"strconv"
	"strings"
)

// 解析器，解析接收到的请求

type Payload struct {
	Data resp.Reply // 接收和发送的数据都叫Reply
	Err  error
}

type readState struct {
	// 解析器的状态
	readingMutiLine   bool     // 是否正在读取多行
	expectedArgsCount int      // 预期的参数个数
	msgType           byte     // 消息类型
	args              [][]byte // 参数列表
	bulkLen           int64
}

func (r *readState) finished() bool {
	// 判断解析器是否完成
	return r.expectedArgsCount > 0 && len(r.args) == r.expectedArgsCount
}

func ParseStream(reader io.Reader) <-chan *Payload {
	// 解析RESP协议流
	// 通过管道输出
	ch := make(chan *Payload)
	go parse0(reader, ch)
	return ch
}

func parse0(reader io.Reader, ch chan<- *Payload) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(string(debug.Stack()))
		}
	}()
	bufReader := bufio.NewReader(reader)
	var state readState
	var err error
	var msg []byte
	for {
		var ioErr bool
		msg, ioErr, err = readLine(bufReader, &state)
		if err != nil {
			if ioErr {
				ch <- &Payload{Err: err}
				close(ch)
				return
			}
			ch <- &Payload{
				Err: err,
			}
			state = readState{}
			continue
		}
		if !state.readingMutiLine {
			// 判断不是多行解析
			if msg[0] == '*' { // *3\r\n
				err := parseMutiBulkHeader(msg, &state)
				if err != nil {
					ch <- &Payload{Err: err}
					state = readState{}
					continue
				}
				if state.expectedArgsCount == 0 {
					ch <- &Payload{Data: &reply.EmptyMutiBulkReply{}}
					state = readState{}
					continue
				}
			} else if msg[0] == '$' { //$3\r\n
				err := parseBulkHeader(msg, &state)
				if err != nil {
					ch <- &Payload{Err: errors.New("protocol err:" + string(msg))}
					state = readState{}
					continue
				}
				if state.bulkLen == -1 {
					ch <- &Payload{Data: &reply.NullBulkReply{}}
					state = readState{}
					continue
				}
			} else {
				lineReply, err := parseSingleLineReply(msg)
				ch <- &Payload{Data: lineReply, Err: err}
				state = readState{}
				continue
			}
		} else {
			err := readBody(msg, &state)
			if err != nil {
				ch <- &Payload{Err: errors.New("protocol err:" + string(msg))}
				state = readState{}
				continue
			}
			if state.finished() {
				var result resp.Reply
				if state.msgType == '*' {
					result = reply.MakeMultiBulkReply(state.args)
				} else if state.msgType == '$' {
					result = reply.MakeMultiBulkReply(state.args) // 这里直接用多行回复包装
				}
				ch <- &Payload{Data: result, Err: nil}
				state = readState{}
			}
		}
	}
	// 解析器的核心函数
	// 读取数据并解析成RESP协议格式
	// 这里可以实现一个简单的RESP解析逻辑
	// 例如读取字节，判断类型，处理多行等
}

// $5\r\n\r\n1\r\n
// *3\r\n$3\r\nSET\r\n$4\r\nlzzy\r\n$7\r\nwelcome\r\n
// *2\r\n$3\r\nGET\r\n$4\r\nlzzy\r\n
// *2\r\n$4\r\nTYPE\r\n$4\r\nlzzy\r\n
// *2\r\n$6\r\nselect\r\n$1\r\n1\r\n
func readLine(bufReader *bufio.Reader, state *readState) ([]byte, bool, error) {
	// 读取一行数据
	// 1. \r\n切分
	// 2. 读取到$+number，严格读取\r\n后面的数据
	// 返回：读取的信息，是否是io错误，错误类型
	var msg []byte
	var err error
	if state.bulkLen == 0 {
		// 没有预设的个数，按照\r\n区分
		msg, err = bufReader.ReadBytes('\n')
		if err != nil {
			return nil, true, err // 其他错误直接返回
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' || msg[len(msg)-1] != '\n' {
			return nil, false, errors.New("protocol error:must end with \\r\\n")
		}
	} else {
		msg = make([]byte, state.bulkLen+2) // +2是为了包含\r\n
		_, err := io.ReadFull(bufReader, msg)
		if err != nil {
			return nil, true, err // 其他错误直接返回
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' || msg[len(msg)-1] != '\n' {
			return nil, false, errors.New("protocol error:invalid bulk format, must end with \\r\\n")
		}
		state.bulkLen = 0 // 读取完毕后重置bulkLen
	}
	return msg, false, nil
}

// *3\r\n$5\r\nhello\r\n$4\r\nlzzy\r\n$7\r\nwelcome\r\n
func parseMutiBulkHeader(msg []byte, state *readState) error {
	// 根据msg的信息，修改readState状态
	var err error
	var expectedLine uint64
	expectedLine, err = strconv.ParseUint(string(msg[1:len(msg)-2]), 10, 32)
	if err != nil {
		return errors.New("parseMutiBulkHeader ParseUint error:" + string(msg))
	}
	if expectedLine == 0 {
		state.expectedArgsCount = 0
		return nil
	} else if expectedLine > 0 {
		state.msgType = msg[0]
		state.readingMutiLine = true
		state.expectedArgsCount = int(expectedLine)
		state.args = make([][]byte, 0, expectedLine)
		return nil
	} else {
		return errors.New("parseMutiBulkHeader expectedLine error:" + string(msg))
	}
}

// $4\r\nPING\r\n
func parseBulkHeader(msg []byte, state *readState) error {
	var err error
	state.bulkLen, err = strconv.ParseInt(string(msg[1:len(msg)-2]), 10, 64)
	if err != nil {
		return errors.New("parseBulkHeader ParseInt error:" + string(msg))
	}
	if state.bulkLen == -1 {
		return nil
	} else if state.bulkLen > 0 {
		state.readingMutiLine = true
		state.msgType = msg[0]
		state.expectedArgsCount = 1
		state.args = make([][]byte, 0, 1)
		return nil
	} else {
		return errors.New("parseBulkHeader bulkLen error:" + string(msg))
	}
}

// +OK -err
func parseSingleLineReply(msg []byte) (resp.Reply, error) {
	str := strings.TrimSuffix(string(msg), "\r\n")
	var result resp.Reply
	switch str[0] {
	case '+':
		result = reply.MakeStatusReply(str[1:])
	case '-':
		result = reply.MakeErrReply(str[1:])
	case ':':
		val, err := strconv.ParseInt(str[1:], 10, 64)
		if err != nil {
			return nil, errors.New("parseSingleLineReply error:" + string(msg))
		}
		result = reply.MakeIntReply(val)
	}
	return result, nil
}

// PING\r\n
// $5\r\n\r\n1\r\n
// $5\r\n\r\n1\r\n$5\r\n\r\n1\r\n$5\r\n\r\n1\r\n
func readBody(msg []byte, state *readState) error {
	var err error
	line := msg[0 : len(msg)-2]
	if line[0] == '$' {
		state.bulkLen, err = strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			return errors.New("readBody error:" + string(msg))
		}
		if state.bulkLen <= 0 {
			state.args = append(state.args, []byte{})
			state.bulkLen = 0
		}
	} else {
		state.args = append(state.args, line)
	}
	return nil
}
