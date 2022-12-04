package cache

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
	// peers 是 HTTPPOOl 类型，实现了 PeerPicker 接口
	peers PeerPicker
}

// Getter 接口的 Get 方法用于根据 key 获取 value
type Getter interface {
	Get(key string) ([]byte, error)
}

// 函数类型
type GetterFunc func(key string) ([]byte, error)

// 函数类型实现 Getter 接口（接口型函数）
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

	// 从缓存中获取到了就直接返回
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	// 获取不到就加载尝试去加载（从其他节点去获取缓存）
	return g.load(key)
}

// 将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer()
// 从远程获取。若是本机节点或失败，则回退到 getLocally()
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
	}

	return g.getLocally(key)
}

// 调用 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中
func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用函数类型的实现的 Get 方法获取值
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

// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
