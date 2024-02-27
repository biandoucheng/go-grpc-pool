package gogrpcpool

import "sync/atomic"

// 引用计算

// 连接的总引用次数加一
func (p *Pool) addConnRefCount() int32 {
	return atomic.AddInt32(&p.refCount, 1)
}

// 连接的总引用次数减一
func (p *Pool) subConnRefCount() int32 {
	ref := atomic.AddInt32(&p.refCount, -1)
	if ref < 0 {
		return atomic.AddInt32(&p.refCount, 1)
	}
	return ref
}

// 检查引用次数
func (p *Pool) chkConnReferd() int32 {
	return atomic.LoadInt32(&p.refCount)
}

// 当前连接的引用总数是否达到了目标比率以上
func (p *Pool) connRefReached() bool {
	return p.chkConnReferd() >= p.opts.MaxRefs*(p.chkConnCount()-p.chkClosingConnCount())/p.opts.NewConnRate
}

// 重置连接的引用次数
func (p *Pool) resetConnRefCount(count int32) {
	atomic.StoreInt32(&p.refCount, count)
}
