package gogrpcpool

import "sync/atomic"

// 连接数配额相关计算

// 索取一个连接额度
func (p *Pool) askConnQuota() bool {
	quota := atomic.AddInt32(&p.connQuota, -1)
	if quota < 0 {
		atomic.AddInt32(&p.connQuota, 1)
		return false
	}
	return true
}

// 归还一个连接额度
func (p *Pool) rbkConnQuota() {
	quota := atomic.AddInt32(&p.connQuota, 1)
	if quota > p.opts.MaxConns {
		atomic.AddInt32(&p.connQuota, -1)
	}
}
