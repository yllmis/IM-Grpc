package configserver

import (
	"errors"

	"github.com/zeromicro/go-zero/core/conf"
)

var ErrNotSetConfig = errors.New("未设置配置信息")

type OnChange func([]byte) error

type ConfigServer interface {
	Build() error
	SetOnChange(OnChange)

	FormJsonBytes() ([]byte, error)
	// Error() error
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

func (c *configServer) MustLoad(v any, onChange OnChange) error {

	if c.configFile == "" && c.ConfigServer == nil {
		return ErrNotSetConfig
	}

	if c.ConfigServer == nil {
		// 使用go-zero默认
		conf.MustLoad(c.configFile, v)
		return nil
	}

	if onChange != nil {
		c.SetOnChange(onChange)
	}

	if err := c.ConfigServer.Build(); err != nil {
		return err
	}

	data, err := c.ConfigServer.FormJsonBytes()
	if err != nil {
		return err
	}

	return LoadFromJsonBytes(data, v)
}

func LoadFromJsonBytes(data []byte, v any) error {
	return conf.LoadFromJsonBytes(data, nil)
}
