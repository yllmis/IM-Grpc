package websocket

import "fmt"

// ResolveAckModes 校验 WS 传输层 ACK 与观测 AckMode 一致。
// 二者不一致时返回错误，调用方必须拒绝启动（见 docs/compatibility-and-rollout.md §5.1）。
//
// 返回：
//   - wsAck：WS 传输层 AckType
//   - obsAck：归一化后的观测 AckMode 字符串
//   - ackObserve：是否允许记录 ack_* 事件（仅 Enabled 且 RigorAck）
func ResolveAckModes(wsAckMode, obsAckMode string, observationEnabled bool) (wsAck AckType, obsAck string, ackObserve bool, err error) {
	wsAck, err = ParseAckType(wsAckMode)
	if err != nil {
		return 0, "", false, err
	}

	obsAck, err = normalizeObsAck(obsAckMode)
	if err != nil {
		return 0, "", false, err
	}

	if wsAck.ToString() != obsAck {
		return 0, "", false, fmt.Errorf("websocket: ack mode mismatch: transport=%s observation=%s", wsAck.ToString(), obsAck)
	}

	// NoAck / OnlyAck 不产生真实客户端 ACK，禁止伪造
	ackObserve = observationEnabled && wsAck == RigorAck
	return wsAck, obsAck, ackObserve, nil
}

func normalizeObsAck(s string) (string, error) {
	switch s {
	case "", "NoAck":
		return "NoAck", nil
	case "OnlyAck":
		return "OnlyAck", nil
	case "RigorAck":
		return "RigorAck", nil
	default:
		return "", fmt.Errorf("websocket: invalid observation AckMode %q", s)
	}
}
