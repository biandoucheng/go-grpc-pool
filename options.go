package gogrpcpool

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

type Options struct {
	Debug            bool              // 开启调试模式之后，会在运行时打印连接使用情况的统计信息
	DescribeDuration time.Duration     // 连接使用情况的打印周期
	CheckPeriod      time.Duration     // 定时清理多出连接的周期
	ConnTimeOut      time.Duration     // 新建连接的超时时间
	Target           string            // grpc 地址
	Dopts            []grpc.DialOption // grpc 拨号选项
	MaxConns         int32             // 最大连接数, -1 = unlimited
	MaxIdleConns     int32             // 最大空闲连接数, min = 1
	MaxRefs          int32             // 每个连接的最大可同时引用的次数
	NewConnRate      int32             // 新连接建立所遵循的指标， 结合 MaxRefs 来确定是否需要建立新连接，当已建立的连接的引用的总次数占它们总的最大可引用次数的 1/NewConnRate 时会尝试建立新的连接;
}

func (o *Options) Dial(tunnel chan<- *Conn) (*Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), o.ConnTimeOut)
	defer cancel()

	grpcconn, err := grpc.DialContext(ctx, o.Target, o.Dopts...)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		conn:        grpcconn,
		ref:         0,
		refMax:      o.MaxRefs,
		closing:     false,
		readying:    false,
		readyTunnel: tunnel,
	}
	go conn.run()

	return conn, nil
}
