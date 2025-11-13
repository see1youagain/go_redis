package resp

type Reply interface {
	ToBytes() []byte // 转换成字节发送
}
