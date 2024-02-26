package gogrpcpool

import (
	"time"

	"google.golang.org/grpc"
)

type Options struct {
	Debug        bool              // debug
	CheckPeriod  time.Duration     // clear period
	Target       string            // target address
	Dopts        []grpc.DialOption // grpc dial options
	MaxIdleConns int32             // max idle connections, min = 1
	MaxConns     int32             // max connections, -1 = unlimited
	MaxRefs      int32             // max reference
	NewByRefRate int32             // 根据连接的引用比例来确定是否新建连接 > 1 正整数; eg: 2 = 1/2 = 50% 即当当前的所有已建立非关闭状态的连接其连接引用率达到 50% 以上才触发新建连接
	PanicOnErr   bool              // panic if error
}

func (o *Options) Dial() (*Conn, error) {
	conn, err := grpc.Dial(o.Target, o.Dopts...)
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn:    conn,
		ref:     0,
		closing: false,
	}, nil
}
