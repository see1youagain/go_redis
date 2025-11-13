package resp

type Connection interface {
	Write([]byte) error // 写入数据
	GetDBIndex() int    // 得到DB索引
	SelectDB(int)       // 切换DB
}
