package gogrpcpool

// 寻求一个可用的连接
func (p *Pool) Acquire() (*Conn, error) {
	// 连接已满,强行复用已有连接
	if p.NoMoreConnect() {
		return p.picker(p.opts.MaxConns, true)
	}

	// 尝试建立新连接
	// 1. 当前连接的引用总数是否达到了目标比率以上，此时新建一个连接
	// 2. 通过原子操作连接配额，来控制并发避免超出最大连接数
	if p.connRefReached() && p.askConnQuota() {
		// 新建连接成功
		p.Lock()
		if conn, err := p.newConn(); err == nil {
			p.addConnRefCount()
			p.Unlock()
			return conn, nil
		}
		p.Unlock()

		// 新建连接失败，强行复用下一个连接
		// 1. 连接资源归还
		//2. 这里不用担心连接引用是否超过最大数,因为连接池越往后的连接引用数一定大于前面的连接引用数
		p.rbkConnQuota()
		return p.picker(p.opts.MaxConns, true)
	}

	// 通常是复用已有的连接
	return p.picker(p.opts.MaxConns, false)
}

// 归还一个连接
// 1. 需要在 defer func 中执行
// 2. 当连接的引用数为0时，说明连接处于准备好的空闲状态
func (p *Pool) Release(conn *Conn) {
	conn.release()
	if p.subConnRefCount() == 0 {
		p.addIdleConnCount()
	}
}

// 从连接池中挑一个
// 1. 挑选的顺序是优先挑选的被引用次数少的连接
// 2. 从connIndex开始到最后一个索引进行查找
// 3. 再从 0 到 connIndex 进行查找
// 4. connIndex 是一个从小到大的值，由于连接有后台关闭和新建的操作所以 connIndex 并不一定能准确找到可用连接，但是他是一个相对准切的近似索引
func (p *Pool) picker(max int32, abs bool) (*Conn, error) {
	p.RLock()
	defer p.RUnlock()

	// 计算索引
	index := p.addConnIndex()
	connCount := p.chkConnCount()

	// 排查后面部分
	for i := index; i < connCount; i++ {
		ref, err := p.conns[i].acquire(max, abs)
		if err != nil {
			continue
		}

		// 引用数为1说明一个连接第一次被使用，意味着空闲连接数少了一个
		if ref == 1 {
			p.subIdleConnCount()
		}

		// 拿到连接后，引用数加1
		p.addConnRefCount()
		return p.conns[i], nil
	}

	// 排查前面部分
	for i := int32(0); i < index; i++ {
		ref, err := p.conns[i].acquire(max, abs)
		if err != nil {
			continue
		}

		// 引用数为1说明一个连接第一次被使用，意味着空闲连接数少了一个
		if ref == 1 {
			p.subIdleConnCount()
		}

		// 拿到连接后，引用数加1
		p.addConnRefCount()
		return p.conns[i], nil
	}

	return nil, ErrNoConnAvailable
}
