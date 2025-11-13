package dict

type Consumer func(key string, val interface{}) bool

// 返回true表示继续遍历，返回false表示停止遍历

type Dict interface {
	Get(key string) (val interface{}, exists bool)
	Len() int                                     // 返回字典的长度
	Put(key string, val interface{}) (result int) // 添加或更新键值对
	PutIfAbsent(key string, val interface{}) (result int)
	PutIfExists(key string, val interface{}) (result int)
	Remove(key string) (result int)
	ForEach(consumer Consumer)
	Keys() []string
	RandomKeys(n int) []string         // 随机获取n个键
	RandomDistinctKeys(n int) []string // 随机获取n个不同的键
	Clear()
}
