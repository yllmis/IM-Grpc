package websocket

import (
	"math"
	"time"
)

const (
	defaultMaxConnectionIdle = time.Duration(math.MaxInt64)
	defaultAckTimeout        = time.Second * 30
	defaultSendErrCount      = 5
)
