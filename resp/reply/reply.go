package reply

import (
	"bytes"
	"go_redis/interface/resp"
	"strconv"
)

var (
	nullBulkReplyBytes = []byte("$-1") // 空字符串
	CRLF               = "\r\n"        // 空字符串
)

type BulkReply struct {
	Arg []byte // "$6\r\nstring\r\n"
}

func (b BulkReply) ToBytes() []byte {
	if len(b.Arg) == 0 {
		return nullBulkReplyBytes
	}
	return []byte("$" + strconv.Itoa(len(b.Arg)) + CRLF + string(b.Arg) + CRLF)
}
func MakeBulkReply(arg []byte) *BulkReply {
	return &BulkReply{Arg: arg}
}

type MultiBulkReply struct {
	Args [][]byte
}

func (m MultiBulkReply) ToBytes() []byte {
	argLen := len(m.Args)
	var buf bytes.Buffer
	buf.WriteString("*" + strconv.Itoa(argLen) + CRLF)
	for _, arg := range m.Args {
		if len(arg) == 0 {
			buf.WriteString(string(nullBulkBytes) + CRLF)
		} else {
			buf.WriteString("$" + strconv.Itoa(len(arg)) + CRLF + string(arg) + CRLF)
		}
	}
	return buf.Bytes()
}
func MakeMultiBulkReply(arg [][]byte) *MultiBulkReply {
	return &MultiBulkReply{Args: arg}
}

type StatusReply struct {
	Status string
}

func MakeStatusReply(status string) *StatusReply {
	return &StatusReply{Status: status}
}
func (r *StatusReply) ToBytes() []byte {
	return []byte("+" + r.Status + CRLF)
}

type IntReply struct {
	Code int64
}

func (r *IntReply) ToBytes() []byte {
	return []byte(":" + strconv.FormatInt(r.Code, 10) + CRLF)
}
func MakeIntReply(code int64) *IntReply {
	return &IntReply{
		Code: code,
	}
}

type ErrorReply interface {
	Error() string
	ToBytes() []byte
}

type StandardErrReply struct {
	Status string
}

func (r *StandardErrReply) ToBytes() []byte {
	return []byte("-" + r.Status + CRLF)
}
func (r *StandardErrReply) Error() string {
	return r.Status
}
func MakeErrReply(status string) *StandardErrReply {
	return &StandardErrReply{
		Status: status,
	}
}

func IsErrReply(reply resp.Reply) bool {
	return reply.ToBytes()[0] == '-'
}
