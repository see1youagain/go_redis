package consistenthash

import (
	"hash/crc32"
	"sort"
)

type HashFunc func(data []byte) uint32

type NodeMap struct {
	// 节点映射
	hashFunc    HashFunc
	nodeHashs   []int
	nodeHashMap map[int]string
}

func NewNodeMap(hashFunc HashFunc) *NodeMap {
	m := &NodeMap{
		hashFunc:    hashFunc,
		nodeHashs:   make([]int, 0),
		nodeHashMap: make(map[int]string),
	}
	if m.hashFunc == nil {
		m.hashFunc = crc32.ChecksumIEEE
	}
	return m
}

func (m *NodeMap) IsEmpty() bool {
	return len(m.nodeHashs) == 0
}

// 只是把新节点加入哈希环，没有处理节点变动时数据迁移的问题
func (m *NodeMap) AddNodes(keys ...string) {
	for _, key := range keys {
		if key == "" {
			continue
		}
		hash := int(m.hashFunc([]byte(key)))
		m.nodeHashs = append(m.nodeHashs, hash)
		m.nodeHashMap[hash] = key
	}
	sort.Ints(m.nodeHashs)
}

// 给定一个key，通过哈希找到应该存储或访问的节点。它保证同一个key总是映射到同一个节点，除非节点变动。
func (m *NodeMap) PickNode(key string) string {
	if m.IsEmpty() {
		return ""
	}
	hash := int(m.hashFunc([]byte(key)))
	idx := sort.Search(len(m.nodeHashs), func(i int) bool { return m.nodeHashs[i] >= hash })
	if idx == len(m.nodeHashs) {
		// 如果大于最后一个hashNode，他就是存在第一个节点中
		idx = 0
	}
	return m.nodeHashMap[m.nodeHashs[idx]]
}
