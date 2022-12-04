package cache

// PeerPicker 是一个节点用来获取自己的 key 的接口
type PeerPicker interface {
	// 根据传入的 key 去选择相应节点的 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是一个节点用来获取远程节点的 key 的接口
type PeerGetter interface {
	// 从对应 group 中查找缓存值
	Get(group string, key string) ([]byte, error)
}
