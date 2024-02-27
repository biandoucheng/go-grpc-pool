package gogrpcpool

// 标记一个连接处于正在关闭状态
// 1. 关闭中的连接不能被再次引用
func (c *Conn) setClosing() {
	c.closing = true
}

// 判断一个连接是否可以被删除
// 1. 连接需要处于关闭中状态
// 2. 连接的引用数必须小于等于0
func (c *Conn) removeAble() bool {
	return c.closing && c.chkRef() <= 0
}

// 关闭这个连接
func (c *Conn) close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
