package dict

import "sync"

type SyncDict struct {
	m sync.Map
}

func (dict *SyncDict) Get(key string) (val interface{}, exists bool) {
	value, ok := dict.m.Load(key)
	return value, ok
}

func (dict *SyncDict) Len() int {
	length := 0
	dict.m.Range(func(key, value interface{}) bool {
		length++
		return true // 继续遍历
	})
	return length
}

func (dict *SyncDict) Put(key string, val interface{}) (result int) {
	_, existed := dict.m.Load(key)
	dict.m.Store(key, val)
	if existed {
		return 0 // 更新操作
	}
	return 1 // 新增操作
}

func (dict *SyncDict) PutIfAbsent(key string, val interface{}) (result int) {
	_, existed := dict.m.LoadOrStore(key, val)
	if existed {
		return 0 // 键已存在，未进行添加
	}
	dict.m.Store(key, val)
	return 1 // 新增操作
}

func (dict *SyncDict) PutIfExists(key string, val interface{}) (result int) {
	_, existed := dict.m.Load(key)
	if !existed {
		return 0 // 键不存在，未进行更新
	}
	dict.m.Store(key, val)
	return 1 // 更新操作
}

func (dict *SyncDict) Remove(key string) (result int) {
	_, existed := dict.m.Load(key)
	if !existed {
		return 0 // 键不存在，未进行删除
	}
	dict.m.Delete(key)
	return 1 // 删除操作
}

func (dict *SyncDict) ForEach(consumer Consumer) {
	dict.m.Range(func(key, value interface{}) bool {
		consumer(key.(string), value)
		return true // 继续遍历
	})
}

func (dict *SyncDict) Keys() []string {
	keys := make([]string, dict.Len())
	index := 0
	dict.m.Range(func(key, value interface{}) bool {
		keys[index] = key.(string)
		index++
		return true // 继续遍历
	})
	return keys
}

func (dict *SyncDict) RandomKeys(n int) []string {
	result := make([]string, n)
	for i := 0; i < n; i++ {
		dict.m.Range(func(key, value interface{}) bool {
			result[i] = key.(string)
			return false
		})
	}
	return result
}

func (dict *SyncDict) RandomDistinctKeys(n int) []string {
	result := make([]string, n)
	index := 0
	dict.m.Range(func(key, value interface{}) bool {
		result[index] = key.(string)
		index++
		if index >= n {
			return false // 达到所需数量，停止遍历
		}
		return true
	})
	return result
}

func (dict *SyncDict) Clear() {
	*dict = *MakeSyncDict()
}

func MakeSyncDict() *SyncDict {
	return &SyncDict{}
}
