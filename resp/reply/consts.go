package reply

type PongReply struct {
	// 回复ping指令
}

var pongBytes = []byte("+PONG\r\n")

// ToBytes 将PongReply转换为字节数组
func (r PongReply) ToBytes() []byte {
	return pongBytes
}

// MakePongReply 创建一个新的PongReply实例
func MakePongReply() *PongReply {
	// 创建一个新的PongReply
	return &PongReply{}
}

type OkReply struct {
	// 回复ping指令
}

var okBytes = []byte("+OK\r\n")

// ToBytes 将PongReply转换为字节数组
func (r OkReply) ToBytes() []byte {
	return okBytes
}

var theOkReply = new(OkReply)

// MakePongReply 创建一个新的PongReply实例
func MakeOkReply() *OkReply {
	// 创建一个新的PongReply
	return theOkReply
}

type NullBulkReply struct {
}

var nullBulkBytes = []byte("$-1\r\n")

func (n NullBulkReply) ToBytes() []byte {
	return nullBulkBytes
}

func MakeNullBulkReply() *NullBulkReply {
	return &NullBulkReply{}
}

type EmptyMutiBulkReply struct {
}

var emptyMutiBulkBytes = []byte("$*0\r\n")

func (n EmptyMutiBulkReply) ToBytes() []byte {
	return emptyMutiBulkBytes
}

func MakeEmptyMutiBulkReply() *EmptyMutiBulkReply {
	return &EmptyMutiBulkReply{}
}

type NoReply struct {
}

var noBytes = []byte("")

func (n NoReply) ToBytes() []byte {
	return noBytes
}

func MakeNoReply() *NoReply {
	return &NoReply{}
}
