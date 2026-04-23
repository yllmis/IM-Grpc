package configserver

import (
	"encoding/json"

	"github.com/HYY-yu/sail-client"
)

type Config struct {
	ETCDEndpoints  string `toml:"etcd_endpoints"` // 逗号分隔的ETCD地址，0.0.0.0:2379,0.0.0.0:12379,0.0.0.0:22379
	ProjectKey     string `toml:"project_key"`
	Namespace      string `toml:"namespace"`
	Configs        string `toml:"configs"`          // 逗号分隔的 config_name.config_type，如：mysql.toml,cfg.json,redis.yaml，空代表不下载任何配置
	ConfigFilePath string `toml:"config_file_path"` // 本地配置文件存放路径，空代表不存储本都配置文件
	LogLevel       string `toml:"log_level"`        // 日志级别(DEBUG\INFO\WARN\ERROR)，默认 WARN
}

type Sail struct {
	*sail.Sail
}

func NewSail(cfg *Config) *Sail {
	s := sail.New(&sail.MetaConfig{
		ETCDEndpoints:  cfg.ETCDEndpoints,
		ProjectKey:     cfg.ProjectKey,
		Namespace:      cfg.Namespace,
		Configs:        cfg.Configs,
		ConfigFilePath: cfg.ConfigFilePath,
		LogLevel:       cfg.LogLevel,
	})
	return &Sail{Sail: s}

}

func (s *Sail) FormJsonBytes() ([]byte, error) {
	if err := s.Pull(); err != nil {
		return nil, err
	}

	v, err := s.MergeVipers() // 将多个配置文件合并成一个viper实例
	if err != nil {
		return nil, err
	}

	data := v.AllSettings() // 获取合并后的配置数据
	return json.Marshal(data)
}

func (s *Sail) Error() error {
	return s.Err()
}
