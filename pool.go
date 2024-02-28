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
	sync.RWMutex

	opts  Options
	conns []*Conn

	connQuota        int32 // 最大连接数配额，新建连接时减一，关闭连接时加一
	connCount        int32 // 当前已建立连接数，用来做真实连接数计算
	connIdleCount    int32 // 当前空闲连接数
	connClosingCount int32 // 正处于关闭中状态的连接数

	refCount    int32      // 连接总的引用计数
	readyTunnel chan *Conn // 就绪连接通道，连接池从这里去就绪的连接，就绪的连接主动将自己推入这个通道
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
			CloseWait:        opts.CloseWait,
			ConnTimeOut:      opts.ConnTimeOut,
			ConnBlock:        opts.ConnBlock,
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

	if pool.opts.CloseWait < time.Second {
		pool.opts.CloseWait = time.Second * 20
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
		conn.setClosing(true)
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

		if _, err := p.newConn(p.opts.ConnBlock); err != nil {
			p.rbkConnQuota()
			continue
		}
	}
}

// 新建连接
func (p *Pool) newConn(block bool) (*Conn, error) {
	p.Lock()
	defer p.Unlock()

	st := time.Now().UTC().UnixMilli()
	conn, err := p.opts.Dial(p.readyTunnel, block)
	if p.opts.Debug {
		log.Printf("newConn: cost %v ms", time.Now().UnixMilli()-st)
	}

	if err != nil {
		return nil, err
	}

	p.conns = append(p.conns, conn)
	p.addConnCount()
	p.addIdleConnCount()
	return conn, nil
}
