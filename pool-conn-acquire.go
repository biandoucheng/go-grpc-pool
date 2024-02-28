package gogrpcpool

import (
	"context"
	"time"
)

// 寻求一个可用的连接
func (p *Pool) Acquire(d time.Duration) (*Conn, error) {
	// 先把引用次数加一 避免并发导致无法在此新建连接
	ref := p.addConnRefCount()

	// 尝试建立新连接
	// 1. 当前连接的引用总数 达到了目标引用占比以上，此时新建一个连接
	// 2. 通过原子操作申请连接配额，来避免并发新建连接导致连接数超出最大限制
	if p.connRefReached(ref) && p.askConnQuota() {
		if _, err := p.newConn(false); err != nil {
			// 连接建立失败 连接额度归还
			p.rbkConnQuota()
		}
	}

	// 选取已建立的连接
	conn, err := p.picker(d)
	if err != nil {
		p.subConnRefCount()
	}
	return conn, err
}

// 释放一个连接的引用数
// 1. 需要在 Acquire 之后，在 defer 中执行，避免忘记执行
// 2. 当连接的引用数为0时，说明连接处于空闲状态，对空闲连接数加一
func (p *Pool) Release(conn *Conn) {
	conn.release()
	if p.subConnRefCount() == 0 {
		p.addIdleConnCount()
	}
}

// 从就绪的连接中选一个使用
func (p *Pool) picker(d time.Duration) (*Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	select {
	case conn := <-p.readyTunnel:
		// 引用数为，说明这个连接刚从空闲状态启用，意味着空闲连接数少了一个
		if conn.acquire() == 1 {
			p.subIdleConnCount()
		}
		return conn, nil
	case <-ctx.Done():
		return nil, ErrWaitConnReadyTimeout
	}
}
