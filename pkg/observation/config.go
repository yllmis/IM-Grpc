package observation

import (
	"fmt"
	"os"
	"strings"
)

// ACK 模式。与 apps/im/ws/websocket.AckType 同名同义。
const (
	AckModeNoAck    = "NoAck"
	AckModeOnlyAck  = "OnlyAck"
	AckModeRigorAck = "RigorAck"
)

// DeliveryObservation 观测开关配置。
// 默认值禁止改变现有业务行为：Enabled=false + AckMode=NoAck。
// 第一版 Enabled / AckMode 仅进程启动时读取，修改必须重启。
type DeliveryObservation struct {
	Enabled       bool   `json:"Enabled"`
	PersistEvents bool   `json:"PersistEvents"`
	AckMode       string `json:"AckMode"`
	InstanceId    string `json:"InstanceId"`
	BufferSize    int    `json:"BufferSize"`
}

// DefaultConfig 返回零行为变化的默认配置。
func DefaultConfig() DeliveryObservation {
	return DeliveryObservation{
		Enabled:       false,
		PersistEvents: true,
		AckMode:       AckModeNoAck,
		BufferSize:    1024,
	}
}

// Normalize 填默认值并解析 InstanceId。
func (c *DeliveryObservation) Normalize() {
	if c.AckMode == "" {
		c.AckMode = AckModeNoAck
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 1024
	}
	if c.InstanceId == "" {
		host, err := os.Hostname()
		if err != nil || host == "" {
			host = "unknown"
		}
		c.InstanceId = host
	}
}

// Validate 校验配置合法性。
// AckMode 必须是合法枚举；Enabled=true 时 PersistEvents 必须为 true（否则无法产生可查证据）。
func (c DeliveryObservation) Validate() error {
	switch c.AckMode {
	case AckModeNoAck, AckModeOnlyAck, AckModeRigorAck:
	default:
		return fmt.Errorf("observation: invalid AckMode %q", c.AckMode)
	}
	if c.Enabled && !c.PersistEvents {
		return fmt.Errorf("observation: Enabled=true requires PersistEvents=true")
	}
	return nil
}

// ParseAckMode 归一化 ACK 模式字符串。
func ParseAckMode(s string) (string, error) {
	switch strings.TrimSpace(s) {
	case AckModeNoAck, "":
		return AckModeNoAck, nil
	case AckModeOnlyAck:
		return AckModeOnlyAck, nil
	case AckModeRigorAck:
		return AckModeRigorAck, nil
	default:
		return "", fmt.Errorf("observation: invalid AckMode %q", s)
	}
}
