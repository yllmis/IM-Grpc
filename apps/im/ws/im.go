package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/IM_System/apps/im/ws/internal/config"
	"github.com/IM_System/apps/im/ws/internal/handler"
	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/pkg/configserver"
	"github.com/IM_System/pkg/constants"

	"github.com/zeromicro/go-zero/core/proc"
)

var configFile = flag.String("f", "etc/dev/im copy.yaml", "the config file")

var wg sync.WaitGroup

func main() {
	flag.Parse()

	var c config.Config

	err := configserver.NewConfigServer(*configFile, configserver.NewSail(&configserver.Config{
		ETCDEndpoints:  "etcd:2379",
		ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
		Namespace:      "im",
		Configs:        "im-ws.yaml",
		ConfigFilePath: "./conf",
		LogLevel:       "DEBUG",
	})).MustLoad(&c, func(bytes []byte) error {
		var c config.Config
		if err := configserver.LoadFromJsonBytes(bytes, &c); err != nil {
			return err
		}

		proc.Shutdown()

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
	if err := c.SetUp(); err != nil {
		panic(err)
	}
	ctx := svc.NewServiceContext(c)
	srv := websocket.NewServer(c.ListenOn,
		websocket.WithAuthentication(handler.NewJwtAuth(ctx)),
		websocket.WithMaxIdleConnectionIdle(1000*time.Second),
		// websocket.WithAck(websocket.RigorAck),
		websocket.WithOnClose(func(uid string) {
			ctx.Redis.HdelCtx(context.Background(), constants.REDIS_ONLINE_USERS, uid)
		}),
	)
	defer srv.Stop()

	handler.RegisterHandlers(srv, ctx)

	fmt.Printf("Starting websocket server at %s...\n", c.ListenOn)
	srv.Start()
}