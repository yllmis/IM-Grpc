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

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
)

var configFile = flag.String("f", "etc/dev/im copy.yaml", "the config file")
var configSource = flag.String("config-source", "sail", "configuration source: sail or local")

var wg sync.WaitGroup

func main() {
	flag.Parse()

	c, err := loadConfig(*configFile, *configSource, func(bytes []byte) error {
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

func loadConfig(path, source string, onChange configserver.OnChange) (config.Config, error) {
	var c config.Config
	switch source {
	case "local":
		if err := conf.Load(path, &c, conf.UseEnv()); err != nil {
			return c, err
		}
		return c, nil
	case "sail":
		return c, configserver.NewConfigServer(path, configserver.NewSail(&configserver.Config{
			ETCDEndpoints:  "etcd:2379",
			ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
			Namespace:      "im",
			Configs:        "im-ws.yaml",
			ConfigFilePath: "./conf",
			LogLevel:       "DEBUG",
		})).MustLoad(&c, onChange)
	default:
		return c, fmt.Errorf("unknown config-source %q: use sail or local", source)
	}
}

func Run(c config.Config) {
	if err := c.SetUp(); err != nil {
		panic(err)
	}

	// ACK 双开关必须一致，不一致拒绝启动（见 compatibility-and-rollout.md §5.1）
	wsAck, obsAck, ackObserve, err := websocket.ResolveAckModes(
		c.AckMode,
		c.DeliveryObservation.AckMode,
		c.DeliveryObservation.Enabled,
	)
	if err != nil {
		panic(err)
	}

	ctx := svc.NewServiceContext(c)
	srv := websocket.NewServer(c.ListenOn,
		websocket.WithAuthentication(handler.NewJwtAuth(ctx)),
		websocket.WithMaxIdleConnectionIdle(1000*time.Second),
		websocket.WithAck(wsAck),
		websocket.WithAckObserve(ackObserve),
		websocket.WithObserver(handler.NewSinkObserver(ctx.ObservationSink, c.DeliveryObservation.InstanceId)),
		websocket.WithOnClose(func(uid string) {
			ctx.Redis.HdelCtx(context.Background(), constants.REDIS_ONLINE_USERS, uid)
		}),
	)
	defer srv.Stop()

	handler.RegisterHandlers(srv, ctx)

	fmt.Printf("Starting websocket server at %s... ack=%s obsAck=%s ackObserve=%v\n",
		c.ListenOn, wsAck.ToString(), obsAck, ackObserve)
	srv.Start()
}
