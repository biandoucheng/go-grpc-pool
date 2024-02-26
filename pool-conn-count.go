package gogrpcpool

import "sync/atomic"

// 连接数计算

// 连接数加一
func (p *Pool) addConnCount() {
	count := atomic.AddInt32(&p.connCount, 1)
	if count > p.opts.MaxConns {
		atomic.AddInt32(&p.connCount, -1)
	}
}

// 连接数减一
func (p *Pool) subConnCount() {
	count := atomic.AddInt32(&p.connCount, -1)
	if count < 0 {
		atomic.AddInt32(&p.connCount, 1)
	}
}

// 重置连接数
func (p *Pool) resetConnCount(count int32) {
	atomic.StoreInt32(&p.connCount, count)
}

// 查询连接数
func (p *Pool) chkConnCount() int32 {
	return atomic.LoadInt32(&p.connCount)
}

// 连接已经建立到最大数目
func (p *Pool) NoMoreConnect() bool {
	return p.chkConnCount() >= p.opts.MaxConns
}

// 空闲连接数加一
func (p *Pool) addIdleConnCount() {
	count := atomic.AddInt32(&p.connIdleCount, 1)
	if count > p.opts.MaxConns {
		atomic.AddInt32(&p.connIdleCount, -1)
	}
}

// 空闲连接数减一
func (p *Pool) subIdleConnCount() {
	count := atomic.AddInt32(&p.connIdleCount, -1)
	if count < 0 {
		atomic.AddInt32(&p.connIdleCount, 1)
	}
}

// 将已建立连接数设置为空闲连接数
func (p *Pool) resetIdleConnCount(count int32) {
	atomic.StoreInt32(&p.connIdleCount, count)
}

// 查询空闲连接数
func (p *Pool) chkIdleConnCount() int32 {
	return atomic.LoadInt32(&p.connIdleCount)
}

// 关闭连接数加一
func (p *Pool) addClosingConnCount() {
	count := atomic.AddInt32(&p.connClosingCount, 1)
	if count > p.opts.MaxConns {
		atomic.AddInt32(&p.connClosingCount, -1)
	}
}

// 关闭连接数减一
func (p *Pool) subClosingConnCount() {
	count := atomic.AddInt32(&p.connClosingCount, -1)
	if count < 0 {
		atomic.AddInt32(&p.connClosingCount, 1)
	}
}

// 重置关闭连接数
func (p *Pool) resetClosingConnCount(count int32) {
	atomic.StoreInt32(&p.connClosingCount, count)
}

// 查询关闭连接数
func (p *Pool) chkClosingConnCount() int32 {
	return atomic.LoadInt32(&p.connClosingCount)
}
