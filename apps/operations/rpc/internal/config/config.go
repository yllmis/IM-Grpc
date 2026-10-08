package config

import (
	"fmt"
	"strings"

	"github.com/IM_System/apps/operations/rpc/internal/faultinject"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mongo struct {
		Url string
		Db  string
	}

	// ServiceAuth 服务身份认证（metadata: x-im-service-token）
	ServiceAuth struct {
		Enable bool
		Token  string
		// ExtraTokens 轮换重叠窗口内的旧 token
		ExtraTokens []string
	}

	// DeliveryObservation 只读反映部署观测配置，用于 GetCapabilities 诚实声明
	DeliveryObservation struct {
		Enabled bool
		AckMode string
	}

	// Query 预算
	Query struct {
		// MaxLimit 单次返回上限，超过则 INVALID_ARGUMENT
		MaxLimit int32
		// DefaultLimit limit<=0 时使用
		DefaultLimit int32
	}

	// FaultInjection 仅用于非生产环境的确定性只读故障注入，默认关闭。
	FaultInjection faultinject.Config `json:",optional"`
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Mongo.Url) == "" || strings.TrimSpace(c.Mongo.Db) == "" {
		return fmt.Errorf("operations query requires Mongo URL and database")
	}
	if !c.ServiceAuth.Enable {
		return fmt.Errorf("operations query service authentication must be enabled")
	}
	if strings.TrimSpace(c.ServiceAuth.Token) == "" {
		return fmt.Errorf("operations query service token is required")
	}
	if c.Query.DefaultLimit <= 0 || c.Query.MaxLimit <= 0 || c.Query.DefaultLimit > c.Query.MaxLimit {
		return fmt.Errorf("operations query limits are invalid")
	}
	if err := c.FaultInjection.Validate(c.Mode); err != nil {
		return err
	}
	return nil
}
