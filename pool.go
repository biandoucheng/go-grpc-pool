package gogrpcpool

import (
	"context"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// 连接池
// 1. 一定要时刻保证连接池约靠后的部分引用数是越小的
type Pool struct {
	sync.RWMutex // lock

	opts  Options // dial options
	conns []*Conn // all connections

	connQuota        int32 // quota of connections,用于确认是否还可以建立新的连接,它可能是不准确的，因为是通过原子加法来做余额判断
	connCount        int32 // current count of connections, should be write locked before counting,用来做真实连接数计算
	connIdleCount    int32 // current count of idle connections
	connClosingCount int32 // current count of closing connections

	refCount    int32      // current count of references
	readyTunnel chan *Conn // ready tunnel
}

// 实例化连接池
func NewPool(opts Options) *Pool {
	if opts.Target == "" {
		log.Fatalf("new Pool Failed: %v", ErrTargetNotAvailable)
	}

	pool := &Pool{
		opts: Options{
			Debug:            opts.Debug,
			DescribeDuration: opts.DescribeDuration,
			CheckPeriod:      opts.CheckPeriod,
			ConnTimeOut:      opts.ConnTimeOut,
			Target:           opts.Target,
			Dopts:            []grpc.DialOption{},
			MaxConns:         opts.MaxConns,
			MaxIdleConns:     opts.MaxIdleConns,
			MaxRefs:          opts.MaxRefs,
			NewConnRate:      opts.NewConnRate,
		},
		readyTunnel: make(chan *Conn, opts.MaxConns),
		connQuota:   opts.MaxConns,
		conns:       []*Conn{},
	}

	if pool.opts.CheckPeriod < time.Second*3 {
		pool.opts.CheckPeriod = time.Second * 3
	}

	if pool.opts.NewConnRate < 2 {
		pool.opts.NewConnRate = 2
	}

	if pool.opts.DescribeDuration <= time.Duration(0) {
		pool.opts.DescribeDuration = time.Second * 1
	}

	pool.opts.Dopts = append(pool.opts.Dopts, opts.Dopts...)

	return pool
}

// 启动
func (p *Pool) Run() {
	p.initConns()

	if p.opts.Debug {
		go p.DescribeTimer()
	}

	go p.idleConnManager()
}

// 关闭连接
func (p *Pool) Close() {
	p.Lock()
	defer p.Unlock()

	// 设置执行超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// 标记所有连接为关闭状态
	for _, conn := range p.conns {
		conn.setClosing()
	}

	// 关闭就绪通道
	close(p.readyTunnel)

	// 定时循环检查连接是否被回收完毕
	tricker := time.NewTicker(time.Second * 2)
	for p.chkConnReferd() > 0 {
		select {
		case <-ctx.Done():
			return
		case <-tricker.C:
		}
	}

	// 关闭所有的连接
	for _, conn := range p.conns {
		conn.close()
	}

	// 清空连接池
	p.conns = []*Conn{}

	// 清空原子计数
	p.resetConnCount(0)
	p.resetIdleConnCount(0)
	p.resetClosingConnCount(0)
	p.resetConnRefCount(0)
}

// 初始化连接，按照最大空闲数建立连接
func (p *Pool) initConns() {
	for i := int32(0); i < p.opts.MaxIdleConns; i++ {
		if !p.askConnQuota() {
			continue
		}

		if _, err := p.newConn(); err != nil {
			p.rbkConnQuota()
			continue
		}
	}
}

// 新建连接
func (p *Pool) newConn() (*Conn, error) {
	p.Lock()
	defer p.Unlock()

	conn, err := p.opts.Dial(p.readyTunnel)
	if err != nil {
		return nil, err
	}

	p.conns = append(p.conns, conn)
	p.addConnCount()
	p.addIdleConnCount()
	return conn, nil
}

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
		}

		// 剩余的连接
		connCount += 1
		if conn.chkRef() == 0 {
			idleCount += 1
		}
		conns = append(conns, conn)
	}

	// 标记下次需要关闭的连接
	shouldClosed := idleCount - p.opts.MaxIdleConns
	for i := 0; i < int(shouldClosed); i++ {
		closeCount += 1
		conns[i].setClosing()
	}

	// 重置连接
	p.conns = conns

	// 重置统计值
	p.resetConnCount(connCount)
	p.resetIdleConnCount(idleCount)
	p.resetClosingConnCount(closeCount)
}
