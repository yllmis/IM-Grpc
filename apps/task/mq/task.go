package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/IM_System/apps/task/mq/internal/config"
	"github.com/IM_System/apps/task/mq/internal/handler"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/pkg/configserver"

	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/dev/task.yaml", "the config file")

var wg sync.WaitGroup

func main() {
	flag.Parse()

	var c config.Config

	err := configserver.NewConfigServer(*configFile, configserver.NewSail(&configserver.Config{
		ETCDEndpoints:  "etcd:2379",
		ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
		Namespace:      "task",
		Configs:        "task-mq.yaml",
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

	listen := handler.NewListen(ctx)

	serviceGroup := service.NewServiceGroup()
	for _, s := range listen.Services() {
		serviceGroup.Add(s)
	}
	fmt.Printf("Starting mqueue at...\n")
	serviceGroup.Start()
}