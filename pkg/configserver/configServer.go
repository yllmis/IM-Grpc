package configserver

import (
	"errors"

	"github.com/zeromicro/go-zero/core/conf"
)

var ErrNotSetConfig = errors.New("未设置配置信息")

type ConfigServer interface {
	FormJsonBytes() ([]byte, error)
	Error() error
}

type configServer struct {
	ConfigServer
	configFile string
}

func NewConfigServer(configFile string, s ConfigServer) *configServer {
	return &configServer{
		ConfigServer: s,
		configFile:   configFile,
	}
}

func (c *configServer) MustLoad(v any) error {
	if c.ConfigServer.Error() != nil {
		return c.ConfigServer.Error()
	}

	if c.configFile == "" && c.ConfigServer == nil {
		return ErrNotSetConfig
	}

	if c.ConfigServer == nil {
		// 使用go-zero默认
		conf.MustLoad(c.configFile, v)
		return nil
	}

	data, err := c.ConfigServer.FormJsonBytes()
	if err != nil {
		return err
	}

	return conf.LoadFromJsonBytes(data, v)
}

func (c *configServer) Error() error {
	return c.ConfigServer.Error()
}
