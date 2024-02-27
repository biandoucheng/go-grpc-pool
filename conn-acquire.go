package gogrpcpool

import (
	"sync/atomic"
)

// 申请一个连接的使用权
// 1. 这个调用应该由连接池来执行，引用后引用数进行原子加一
// 2. 使用完毕后，引用者应该调用release进行原子减一
func (c *Conn) acquire() int32 {
	ref := c.addConnRef()
	c.unsetReady()
	return ref
}

// 释放一个连接的使用权
// 1. 对 ref 进行原子减一
func (c *Conn) release() int32 {
	return c.subConnRef()
}

// 连接的引用数加一
func (c *Conn) addConnRef() int32 {
	return atomic.AddInt32(&c.ref, 1)
}

// 连接的引用数加一减一
func (c *Conn) subConnRef() int32 {
	ref := atomic.AddInt32(&c.ref, -1)
	if ref < 0 {
		return atomic.AddInt32(&c.ref, 1)
	}
	return ref
}

// 检查连接引用数
func (c *Conn) chkRef() int32 {
	ref := atomic.LoadInt32(&c.ref)
	if ref < 0 {
		ref = 0
	}
	return ref
}

// 判断连接引用数是否已经达到最大
func (c *Conn) isMaxRef() bool {
	return c.chkRef() >= c.refMax
}
