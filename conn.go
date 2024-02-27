package gogrpcpool

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
)

/*
grpc 客户端连接管理

1. 实例化时设置 ref 为0，每次引用时加一，释放时减一
2. 连接被建立时，需要启动一个goroutine来运行 run() 方法，该方法会实时检测连接的状态，来实时将可用状态下的连接推入到 readyTunnel 中
3. 当 closing = true 时，连接将不允许再被引用，也就不能够再推到 readyTunnel 中，也意味着 ref 的值不会再增加
4. 当 closing = true 且 ref为0 时, 在Pool中会被 idleConnManager 关闭和删除
5. 当 ref >= refMax 时，连接也将不能够再被引用，也就不能够再推到 readyTunnel 中，但是 run 方法会每隔1ms进行一次ref检测，当ref < refMax 时，连接将再次被推入到 readyTunnel 中
*/
type Conn struct {
	// grpc ClientConn
	conn *grpc.ClientConn

	// 连接的引用次数， 每 acquire 一次加一，连接归还时减一
	ref    int32
	refMax int32

	// 关闭状态, 当连接需要准备关闭时，将其设置为true，之后连接将不能够再推到 就绪管道 readyTunnel 中
	closing bool
	// 就绪状态, 当连接被成功推入就绪管道 readyTunnel 中时，被设定为 true, 当连接被取用时，重置为false
	readying bool

	// 就绪管道，该管道是对 Pool 中的 readyTunnel 的引用
	// 在Pool中每当需要连接时，会通过 <-readyTunnel 取用连接
	readyTunnel chan<- *Conn
}

func (c *Conn) run() {
	for {
		if c.closing {
			break
		}

		// 连接的引用次数满了 或者 连接已经处于就绪状态则等待 则睡眠等待
		if c.isMaxRef() || c.readying {
			time.Sleep(time.Millisecond * 1)
			continue
		}

		c.readyTunnel <- c
		c.setReady()
	}
}

// 引用grpc客户端连接
func (c *Conn) Refer() *grpc.ClientConn {
	return c.conn
}

// 描述信息
func (c *Conn) Describe() string {
	return fmt.Sprintf("ref: %v, closing: %v, readying: %v", c.chkRef(), c.closing, c.readying)
}
