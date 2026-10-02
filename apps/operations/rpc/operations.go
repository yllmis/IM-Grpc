package main

import (
	"flag"
	"fmt"

	"github.com/IM_System/apps/operations/rpc/internal/config"
	"github.com/IM_System/apps/operations/rpc/internal/interceptor"
	"github.com/IM_System/apps/operations/rpc/internal/server"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/pkg/interceptor/rpcserver"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/dev/operations.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	// 本地/测试可直接读文件；生产可切换 configserver（Sail+etcd）与其它服务一致
	if err := conf.Load(*configFile, &c, conf.UseEnv()); err != nil {
		panic(err)
	}
	if err := c.Validate(); err != nil {
		panic(err)
	}

	Run(c)
}

func Run(c config.Config) {
	ctx := svc.NewServiceContext(c)

	auth := interceptor.NewAuth(c.ServiceAuth.Enable, append([]string{c.ServiceAuth.Token}, c.ServiceAuth.ExtraTokens...)...)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		operations.RegisterOperationsQueryServer(grpcServer, server.NewOperationsServer(ctx))
		reflection.Register(grpcServer)
	})
	// 服务身份在 interceptor 校验；错误码映射沿用 LoginInterceptorfunc
	s.AddUnaryInterceptors(auth.UnaryInterceptor, rpcserver.LoginInterceptorfunc)

	defer s.Stop()

	fmt.Printf("Starting operations.rpc at %s (auth=%v)...\n", c.ListenOn, c.ServiceAuth.Enable)
	s.Start()
}
