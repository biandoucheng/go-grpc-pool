package gogrpcpool

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// debug 打印
func (p *Pool) DescribeTimer() {
	tricker := time.NewTicker(p.opts.DescribeDuration)
	for {
		<-tricker.C
		fmt.Println(p.Describe())
	}
}

// 输出连接池状态
func (p *Pool) Describe() string {
	summary := fmt.Sprintf("Pool{connCount:%d, refCount:%d, connIdleCount:%d, connClosingCount:%d}\nConns:\n",
		atomic.LoadInt32(&p.connCount),
		atomic.LoadInt32(&p.refCount),
		atomic.LoadInt32(&p.connIdleCount),
		atomic.LoadInt32(&p.connClosingCount))

	conns := []string{}
	p.RLock()
	defer p.RUnlock()
	for _, conn := range p.conns {
		conns = append(conns, conn.Describe())
	}

	return summary + strings.Join(conns, "\n")
}
