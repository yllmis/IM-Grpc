package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/IM_System/apps/user/rpc/internal/config"
	"github.com/IM_System/apps/user/rpc/internal/server"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/configserver"
	"github.com/IM_System/pkg/interceptor/rpcserver"

	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/dev/user.yaml", "the config file")

var wg sync.WaitGroup

var grpcSvr *grpc.Server
var grpcSvrMu sync.Mutex

func main() {
	flag.Parse()

	var c config.Config

	err := configserver.NewConfigServer(*configFile, configserver.NewSail(&configserver.Config{
		ETCDEndpoints:  "etcd:2379",
		ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
		Namespace:      "user",
		Configs:        "user-rpc.yaml",
		ConfigFilePath: "./conf",
		LogLevel:       "DEBUG",
	})).MustLoad(&c, func(bytes []byte) error {
		var c config.Config
		if err := configserver.LoadFromJsonBytes(bytes, &c); err != nil {
			return err
		}

		grpcSvrMu.Lock()
		current := grpcSvr
		grpcSvrMu.Unlock()
		if current != nil {
			current.GracefulStop() // 结束旧服务，释放监听端口
		}

		fmt.Println("更新后的配置", c)
		wg.Add(1)
		go func(c config.Config) {
			defer wg.Done()
			Run(c)
		}(c)

		return nil
	})
	if err != nil {
		panic(err)
	}

	wg.Add(1)
	go func(c config.Config) {
		defer wg.Done()
		Run(c)
	}(c)

	wg.Wait()

}

func Run(c config.Config) {
	ctx := svc.NewServiceContext(c)

	if err := ctx.SetRootToken(); err != nil {
		panic(err)
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		grpcSvrMu.Lock()
		grpcSvr = grpcServer
		grpcSvrMu.Unlock()

		user.RegisterUserServer(grpcServer, server.NewUserServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(rpcserver.LoginInterceptorfunc)
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
