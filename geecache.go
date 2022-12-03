package geecache

import (
	"fmt"
	"log"
	"sync"
)

// 缓存的命名空间
type Group struct {
	// 缓存的名字
	name string
	// 缓存未命中时获取源数据的回调
	getter Getter
	// 自己实现的LRU并发缓存
	mainCache cache
}

// Getter 接口的 Get 方法用于根据 key 获取 value
type Getter interface {
	Get(key string) ([]byte, error)
}

// 函数式接口
type GetterFunc func(key string) ([]byte, error)

// 实现 Getter 接口
func (f GetterFunc) Get(key string) ([]byte, error) {
	// 调用自己
	return f(key)
}

var (
	// 互斥锁
	mu sync.RWMutex
	//存放 Group 的集合
	groups = make(map[string]*Group)
)

// 实例化Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	// 加锁保证并发顺序
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// 返回最先被创建的 Group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 根据 key 获取 cache 中的 value
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 从缓冲中获取到了就直接返回
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	// 获取不到就加载尝试去加载（从其他节点去获取缓存），当前仅实现了单机
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

// 调用 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
