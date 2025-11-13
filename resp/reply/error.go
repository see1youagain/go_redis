package reply

type UnknowErrReply struct {
}

var unknownErrBytes = []byte("-Err unknown\r\n")

func (u UnknowErrReply) Error() string {
	return "ERR unknown error"
}
func (u UnknowErrReply) ToBytes() []byte {
	return unknownErrBytes
}

type ArgNumErrReply struct {
	Cmd string // 返回指令
}

func (a ArgNumErrReply) Error() string {
	return "-ERR wrong number of arguments for '" + a.Cmd + "'commend"
}
func (a *ArgNumErrReply) ToBytes() []byte {
	return []byte("-ERR wrong number of arguments for '" + a.Cmd + "'commend\r\n")
}
func MakeArgNumErrReply(cmd string) *ArgNumErrReply {
	return &ArgNumErrReply{
		Cmd: cmd,
	}
}

// 语法错误
type SyntaxErrReply struct{}

var syntaxErrBytes = []byte("-Err syntax error\r\n")
var theSyntaxErrReply = &SyntaxErrReply{}

func MakeSyntaxErrReply() *SyntaxErrReply {
	return theSyntaxErrReply
}
func (s *SyntaxErrReply) ToBytes() []byte {
	return syntaxErrBytes
}
func (s *SyntaxErrReply) Error() string {
	return "ERR syntax error"
}

type WrongTypeErrReply struct{}

var wrongTypeErrBytes = []byte("-WRONGTYPE Operation against a key holding the wrong kind of value\r\n")
var theWrongTypeErrReply = &WrongTypeErrReply{}

func MakeWrongTypeErrReply() *WrongTypeErrReply {
	return theWrongTypeErrReply
}
func (s *WrongTypeErrReply) ToBytes() []byte {
	return wrongTypeErrBytes
}
func (s *WrongTypeErrReply) Error() string {
	return "-WRONGTYPE Operation against a key holding the wrong kind of value"
}

type ProtocolErrReply struct {
	Msg string
}

func (r *ProtocolErrReply) ToBytes() []byte {
	return []byte("-ERR Protocol error: '" + r.Msg + "'\r\n")
}
func (r *ProtocolErrReply) Error() string {
	return "-ERR Protocol error: '" + r.Msg + "'"
}
