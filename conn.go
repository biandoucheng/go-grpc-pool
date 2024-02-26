package gogrpcpool

import (
	"fmt"
	"sync/atomic"

	"google.golang.org/grpc"
)

/*
grpc 客户端连接管理

1. 实例化时设置 ref 为0，每次引用时加一，释放时减一
2. 后台连接数管理器巡查连接数量时，当空闲连接数超过目标值的时候 且 ref小于等于0时，标记为 notAllowed
3. 获取连接时，需要传入允许的最大引用次数 maxRef，当ref大于等于 maxRef 时，返回 ErrConnTooManyReference
*/
type Conn struct {
	ref     int32            // allow reference count
	closing bool             // allow close connection
	conn    *grpc.ClientConn // grpc *ClientConn
}

// 引用grpc客户端连接
func (c *Conn) Refer() *grpc.ClientConn {
	return c.conn
}

// 申请一个连接的使用权
// 1. 对比每个连接的最大引用次数，如果超过则返回错误
// 2. 如果传入 abd = true 说明连接新建数目已达上限，此时会强行复用当下连接
func (c *Conn) acquire(max int32, abs bool) (int32, error) {
	if c.closing {
		return -1, ErrConnIsClosing
	}

	if abs {
		_ref := c.addConnRef()
		return _ref, nil
	}

	_ref := c.addConnRef()
	if _ref > max {
		return c.subConnRef(), ErrConnTooManyReference
	}

	return _ref, nil
}

// 增加一个连接的引用数
func (c *Conn) addConnRef() int32 {
	return atomic.AddInt32(&c.ref, 1)
}

// 减少一个连接的引用数
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

// 释放一个连接的使用权
// 1. 这个调用应该有连接池来执行
func (c *Conn) release() int32 {
	return c.subConnRef()
}

// 标记一个连接处于正在关闭状态
// 1. 关闭中的连接不能被再次引用
func (c *Conn) setClosing() {
	c.closing = true
}

// 判断一个连接是否可以被删除
func (c *Conn) removeAble() bool {
	return c.closing && c.chkRef() <= 0
}

// 关闭一个连接
func (c *Conn) close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Conn) Describe() string {
	return fmt.Sprintf("ref: %v, closing: %v", atomic.LoadInt32(&c.ref), c.closing)
}
