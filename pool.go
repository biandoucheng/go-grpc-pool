package gogrpcpool

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
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

	connIndex int32 // next index to acquire
	refCount  int32 // current count of references
}

// 实例化连接池
func NewPool(opts Options) *Pool {
	pool := &Pool{
		opts: Options{
			Debug:        opts.Debug,
			CheckPeriod:  opts.CheckPeriod,
			Target:       opts.Target,
			Dopts:        []grpc.DialOption{},
			MaxIdleConns: opts.MaxIdleConns,
			MaxConns:     opts.MaxConns,
			MaxRefs:      opts.MaxRefs,
			NewByRefRate: opts.NewByRefRate,
			PanicOnErr:   opts.PanicOnErr,
		},
		connQuota: opts.MaxConns,
		conns:     []*Conn{},
		connIndex: -1,
	}

	if pool.opts.CheckPeriod < time.Second*3 {
		pool.opts.CheckPeriod = time.Second * 3
	}

	if pool.opts.NewByRefRate < 2 {
		pool.opts.NewByRefRate = 2
	}

	if opts.Target == "" {
		log.Fatalf("new Pool Failed: %v", ErrTargetNotAvailable)
	}
	pool.opts.Dopts = append(pool.opts.Dopts, opts.Dopts...)

	return pool
}

// 启动
func (p *Pool) Run() {
	// 初始化空闲连接
	p.initIdleConns()

	// 启动连接数控制线程
	go p.idleConnManager()

	// debug 打印
	if p.opts.Debug {
		go p.DescribeTimer()
	}
}

// 初始化，按照最大空闲数建立连接
// 1. 只能在初始化时调用一次
func (p *Pool) initIdleConns() {
	for i := int32(0); i < p.opts.MaxIdleConns; i++ {
		if !p.askConnQuota() {
			continue
		}

		if _, err := p.newConn(); err != nil {
			p.rbkConnQuota()
			continue
		}
	}
	p.resetIdleConnCount(p.chkConnCount())
}

// 新建连接
func (p *Pool) newConn() (*Conn, error) {
	conn, err := p.opts.Dial()
	if err != nil {
		return nil, err
	}

	p.conns = append(p.conns, conn)
	atomic.AddInt32(&p.connCount, 1)
	return conn, nil
}

// 连接管理地址
func (p *Pool) idleConnManager() {
	tricker := time.NewTicker(p.opts.CheckPeriod)
	for {
		<-tricker.C
		p.reset()
	}
}

// 重置连接池
// 1. 重置连接池时，任何索取连接的操作都不允许
func (p *Pool) reset() {
	p.Lock()
	defer p.Unlock()

	// 重置索引
	newIdx := int32(-1)
	connIdx := int(p.connIndex) % int(atomic.LoadInt32(&p.connCount))

	// 重置连接统计
	idleCount := int32(0)
	connCount := int32(0)

	// 重置连接池
	conns := []*Conn{}

	// 关闭可以关闭的连接
	for idx, conn := range p.conns {
		// 移除需要关闭的连接
		if conn.removeAble() {
			conn.close()
			p.subClosingConnCount()
			p.rbkConnQuota()
			continue
		}

		// 确定新的索引
		if idx <= connIdx {
			newIdx += 1
		}

		// 保留的连接
		connCount += 1
		if atomic.LoadInt32(&conn.ref) == 0 {
			idleCount += 1
		}
		conns = append(conns, conn)
	}

	// 标记下次需要关闭的连接
	shouldClosed := idleCount - p.opts.MaxIdleConns
	for i := 0; i < int(shouldClosed); i++ {
		p.addClosingConnCount()
		conns[i].setClosing()
	}

	// 重置连接
	p.conns = conns

	// 重置索引
	if int(newIdx+1) > len(p.conns) {
		newIdx = int32(len(p.conns) - 1)
	}
	p.resetConnIndex(newIdx)

	// 重置统计值
	p.resetConnCount(connCount)
	p.resetIdleConnCount(idleCount)

	if shouldClosed < 0 {
		shouldClosed = 0
	}
	p.resetClosingConnCount(shouldClosed)
}

// debug 打印
func (p *Pool) DescribeTimer() {
	tricker := time.NewTicker(time.Millisecond * 500)
	for {
		<-tricker.C
		fmt.Println(p.Describe())
	}
}

// 输出连接池状态
func (p *Pool) Describe() string {
	summary := fmt.Sprintf("Pool{connCount:%d, connIdleCount:%d, connClosingCount:%d, connIndex:%d, refCount:%d}\nConns:\n",
		atomic.LoadInt32(&p.connCount),
		atomic.LoadInt32(&p.connIdleCount),
		atomic.LoadInt32(&p.connClosingCount),
		atomic.LoadInt32(&p.connIndex),
		atomic.LoadInt32(&p.refCount))

	conns := []string{}
	p.RLock()
	defer p.RUnlock()
	for _, conn := range p.conns {
		conns = append(conns, conn.Describe())
	}

	return summary + strings.Join(conns, "\n")
}
