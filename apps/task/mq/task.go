package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/IM_System/apps/task/mq/internal/config"
	"github.com/IM_System/apps/task/mq/internal/handler"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/pkg/configserver"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/dev/task.yaml", "the config file")
var configSource = flag.String("config-source", "sail", "configuration source: sail or local")
var checkConfig = flag.Bool("check-config", false, "validate configuration and exit without starting consumers")

var wg sync.WaitGroup

func main() {
	flag.Parse()

	c, err := loadTaskConfig(*configFile, *configSource, func(bytes []byte) error {
		var c config.Config
		if err := configserver.LoadFromJsonBytes(bytes, &c); err != nil {
			return err
		}

		if err := c.Validate(); err != nil {
			return err
		}
		proc.Shutdown()

		printConfigSummary("sail", c)
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
	printConfigSummary(*configSource, c)
	if *checkConfig {
		return
	}

	wg.Add(1)
	go func(c config.Config) {
		defer wg.Done()
		Run(c)
	}(c)

	wg.Wait()
}

func loadTaskConfig(path, source string, onChange configserver.OnChange) (config.Config, error) {
	var c config.Config
	var err error
	switch source {
	case "local":
		err = conf.Load(path, &c, conf.UseEnv())
	case "sail":
		err = configserver.NewConfigServer(path, configserver.NewSail(&configserver.Config{
			ETCDEndpoints:  "etcd:2379",
			ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
			Namespace:      "task",
			Configs:        "task-mq.yaml",
			ConfigFilePath: "./conf",
			LogLevel:       "DEBUG",
		})).MustLoad(&c, onChange)
	default:
		return c, fmt.Errorf("unknown config-source %q: use sail or local", source)
	}
	if err != nil {
		return c, err
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

func printConfigSummary(source string, c config.Config) {
	// Only print routing information; connection URLs and credentials are omitted.
	fmt.Printf("config source=%s name=%s mongo_db=%s chat_topic=%s chat_group=%s read_topic=%s read_group=%s rate_limit_enabled=%t global_rate=%d instances=%d burst=%d retry_enabled=%t monitoring_enabled=%t\n",
		source, c.Name, c.Mongo.Db, c.MsgChatTransfer.Topic, c.MsgChatTransfer.Group,
		c.MsgReadTransfer.Topic, c.MsgReadTransfer.Group,
		c.MessageProcessingRateLimit.Enabled, c.MessageProcessingRateLimit.GlobalMessagesPerSecond,
		c.MessageProcessingRateLimit.ExpectedInstances, c.MessageProcessingRateLimit.Burst,
		c.MessageRetry.Enabled, c.KafkaMonitoring.Enabled)
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
