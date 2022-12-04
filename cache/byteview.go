package cache

// 一个 ByteView 是一个不可变的 byte 数组
type ByteView struct {
	// 使用 byte 是为了支持任意的数据类型，如字符串或图片
	b []byte
}

// 实现 Value 接口，即实现Len()方法。返回 byte 的长度
func (v ByteView) Len() int {
	return len(v.b)
}

// 以字节数组的形式返回 ByteView 的拷贝（只读，以拷贝的形式返回）
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 返回 string 类型的数据
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
