package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/im/rpc/internal/config"
	"github.com/IM_System/apps/im/rpc/internal/server"
	"github.com/IM_System/apps/im/rpc/internal/svc"
	"github.com/IM_System/pkg/configserver"
	"github.com/IM_System/pkg/interceptor/rpcserver"
	"github.com/IM_System/pkg/serviceauth"

	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/im.yaml", "the config file")

var wg sync.WaitGroup

func main() {
	flag.Parse()

	var c config.Config

	err := configserver.NewConfigServer(*configFile, configserver.NewSail(&configserver.Config{
		ETCDEndpoints:  "etcd:2379",
		ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
		Namespace:      "im",
		Configs:        "im-rpc.yaml",
		ConfigFilePath: "./conf",
		LogLevel:       "DEBUG",
	})).MustLoad(&c, func(bytes []byte) error {
		var c config.Config
		if err := configserver.LoadFromJsonBytes(bytes, &c); err != nil {
			return err
		}

		proc.Shutdown()

		fmt.Printf("更新 im-rpc 配置: listen=%s, read-query-enabled=%v\n", c.ListenOn, c.ReadQueryAuth.Enable)
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
	if err := c.ReadQueryAuth.Validate(); err != nil {
		panic(err)
	}
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		im.RegisterImServer(grpcServer, server.NewImServer(ctx))
		im.RegisterMessageQueryServer(grpcServer, server.NewMessageQueryServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(serviceauth.Guard(c.ReadQueryAuth, "im.MessageQuery"), rpcserver.LoginInterceptorfunc)

	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
