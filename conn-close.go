package gogrpcpool

import "time"

// 标记一个连接处于正在关闭状态
// 1. 关闭中的连接不能被再次引用
// 2. 只有最后一次被引用的时间超过了目标时长且引用个数为0，才能被设置为关闭中
func (c *Conn) setClosing(abs bool) bool {
	if !abs && c.chkRef() > 0 || !c.longTimeNotUse() {
		return false
	}
	c.closing = true
	return true
}

// 长时间未使用
func (c *Conn) longTimeNotUse() bool {
	return c.lastReferAt.Add(c.closeWait).Before(time.Now())
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
