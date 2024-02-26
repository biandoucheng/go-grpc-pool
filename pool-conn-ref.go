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
		atomic.AddInt32(&p.refCount, 1)
		ref += 1
	}
	return ref
}

// 检查引用次数
func (p *Pool) chkConnReferd() int32 {
	return atomic.LoadInt32(&p.refCount)
}

// 当前连接的引用总数是否达到了目标比率以上
func (p *Pool) connRefReached() bool {
	return p.chkConnReferd() >= p.opts.MaxRefs*(p.chkConnCount()-p.chkClosingConnCount())/p.opts.NewByRefRate
}

// 索引步进1
func (p *Pool) addConnIndex() int32 {
	return atomic.AddInt32(&p.connIndex, 1)
}

// 索引步进-1
func (p *Pool) subConnIndex() int32 {
	idx := atomic.AddInt32(&p.connIndex, -1)
	if idx < -1 {
		atomic.AddInt32(&p.connIndex, 1)
		idx += 1
	}
	return idx
}

// 重置索引
func (p *Pool) resetConnIndex(idx int32) {
	atomic.StoreInt32(&p.connIndex, idx)
}
