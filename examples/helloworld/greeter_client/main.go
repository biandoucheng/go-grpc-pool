package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	grpcpool "github.com/biandoucheng/go-grpc-pool"
	pb "github.com/biandoucheng/go-grpc-pool/examples/helloworld/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultName = "world"
)

var (
	addr         = flag.String("addr", "localhost:50051", "the address to connect to")
	name         = flag.String("name", defaultName, "Name to greet")
	grpcConnPool *grpcpool.Pool
)

func init() {
	grpcConnPool = grpcpool.NewPool(grpcpool.Options{
		Debug:        true,
		CheckPeriod:  time.Second * 10,
		Target:       *addr,
		Dopts:        []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
		MaxConns:     30,
		MaxIdleConns: 10,
		MaxRefs:      10,
	})
	grpcConnPool.Run()
}

func sayHello(pool *grpcpool.Pool, n int, tm time.Duration) {
	// 获取连接
	conn, err := pool.Acquire()
	if err != nil {
		log.Printf("Call %d: could not acquire conn: %v", n, err)
		return
	}
	defer pool.Release(conn)

	// 发起grpc请求
	c := pb.NewGreeterClient(conn.Refer())
	ctx, cancel := context.WithTimeout(context.Background(), tm)
	defer cancel()
	_, err = c.SayHello(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		log.Printf("Call %d: could not greet: %v", n, err)
		return
	}
}

func main() {
	flag.Parse()

	tricker := time.NewTicker(time.Second * 2)

	for {

		wg := sync.WaitGroup{}
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				sayHello(grpcConnPool, n, time.Millisecond*10)
			}(i)
		}
		wg.Wait()

		fmt.Println(time.Now())
		<-tricker.C
	}
}
