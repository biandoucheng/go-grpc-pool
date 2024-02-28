package gogrpcpool

import "time"

// 空闲连接数管理
func (p *Pool) idleConnManager() {
	tricker := time.NewTicker(p.opts.CheckPeriod)
	for {
		<-tricker.C
		p.reset()
	}
}

// 重置连接池
func (p *Pool) reset() {
	p.Lock()
	defer p.Unlock()

	idleCount := int32(0)
	connCount := int32(0)
	closeCount := int32(0)
	conns := []*Conn{}

	for _, conn := range p.conns {
		// 移除需要关闭的连接
		if conn.removeAble() {
			conn.close()
			p.rbkConnQuota()
			continue
		}

		// 统计仍处于待关闭状态的
		if conn.closing {
			closeCount += 1
		} else {
			// 统计空闲连接数
			if conn.chkRef() == 0 {
				idleCount += 1
			}
		}

		// 剩余的连接
		connCount += 1
		conns = append(conns, conn)
	}

	// 标记下次需要关闭的连接
	if shouldClosed := idleCount - p.opts.MaxIdleConns; shouldClosed > 0 {
		for i, conn := range conns {
			if shouldClosed <= 0 {
				break
			}

			if conn.closing {
				continue
			}

			// 这里尝试设置连接状态为关闭中
			if conns[i].setClosing(false) {
				closeCount += 1
				shouldClosed -= 1
			}
		}
	}

	// 重置连接
	p.conns = conns

	// 重置统计值
	p.resetConnCount(connCount)
	p.resetIdleConnCount(idleCount)
	p.resetClosingConnCount(closeCount)
}
