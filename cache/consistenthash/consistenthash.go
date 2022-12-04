package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 函数类型，将 byte 转换成 uint32 类型
type Hash func(data []byte) uint32

// Map 容器
type Map struct {
	// Hash函数
	hash Hash
	// 虚拟节点倍数
	replicas int
	// 哈希环
	keys []int
	// 虚拟节点与真实节点的映射表。键是虚拟节点的哈希值，值是真实节点的名称
	hashMap map[int]string
}

// 实例化 Map，允许自定义哈希函数和虚拟节点倍数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加节点到容器中
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 添加虚拟节点
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	// 对环上的哈希值排序
	sort.Ints(m.keys)
}

// 从容器中获取出离 key 最近的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// 二叉搜索节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}

// 从哈希表和哈希环中移除节点
func (m *Map) Remove(key string) {
	for i := 0; i < m.replicas; i++ {
		hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
		idx := sort.SearchInts(m.keys, hash)
		m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
		delete(m.hashMap, hash)
	}
}
