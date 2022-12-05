package singleflight

import "sync"

// call 代表正在进行中或已经结束的请求
type call struct {
	// 避免重入
	wg  sync.WaitGroup
	val interface{}
	err error
}

// 管理不同 key 的请求（call）
type Group struct {
	mu sync.Mutex       // 保护 m 不被并发读写
	m  map[string]*call // 懒初始化，提高内存的使用效率
}

// 针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，
// 等待 fn 调用结束了，返回返回值或错误。
// 使用singleflight，第一个get(key)请求到来时，singleflight会记录当前key正在被处理，
// 后续的请求只需要等待第一个请求处理完成，取返回值即可。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	// 加锁防止 g.m 的并发读写问题
	g.mu.Lock()
	// 初始化 map
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 获取call
	if c, ok := g.m[key]; ok {
		// 能获取到值就可以解锁
		g.mu.Unlock()
		// 如果请求正在进行中，则阻塞至等待组的值为0
		c.wg.Wait()
		// 直接返回结果
		return c.val, c.err
	}
	c := new(call)
	// 发起请求前让等待组加一
	c.wg.Add(1)
	// 添加到 call 表中，代表 key 已经有相应的请求
	g.m[key] = c
	// g.m 没有并发读写问题了就可以解锁
	g.mu.Unlock()

	// 调用 fn，发起请求，这时其他请求都会进入 if 判断中去等待
	c.val, c.err = fn()
	// 请求结束让等待组减一
	c.wg.Done()
	// 加锁解决并发读写问题
	g.mu.Lock()
	// 删掉数据，不需要一直保存，仅是为了解决缓存击穿的问题
	delete(g.m, key)
	// 删完数据解锁
	g.mu.Unlock()

	return c.val, c.err
}
