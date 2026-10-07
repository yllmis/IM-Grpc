package types

import (
	"context"
	"errors"
	"strings"

	"github.com/IM_System/apps/operations/rpc/operationsmodels"
	"github.com/IM_System/pkg/observation"
	"github.com/IM_System/pkg/readquery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gRPC 错误映射契约见 docs/operations-query.md §3.4。
// 核心红线：查询成功无数据 ≠ 查询失败；超时禁止翻译成 found=false。

func InvalidArgument(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

func PermissionDenied(msg string) error {
	return status.Error(codes.PermissionDenied, msg)
}

func Unimplemented(msg string) error {
	return status.Error(codes.Unimplemented, msg)
}

func Internal(msg string) error {
	return status.Error(codes.Internal, msg)
}

// MapQueryError 将底层查询错误映射为 gRPC 状态。
// mongo: no documents 不应走到这里（调用方处理 found=false）。
func MapQueryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, operationsmodels.ErrNotFound) {
		// 调用方应转成空结果，而不是错误
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return status.Error(codes.DeadlineExceeded, "query deadline exceeded")
	}

	// 保留上游 gRPC 的 PermissionDenied/Unavailable 等语义。
	return readquery.MapError(err)
}

// ValidateMessageID 24hex ObjectID。
func ValidateMessageID(id string) error {
	if strings.TrimSpace(id) == "" {
		return InvalidArgument("messageId is required")
	}
	if _, ok := observation.ParseObjectIDHex(id); !ok {
		return InvalidArgument("messageId must be 24hex ObjectID")
	}
	return nil
}

// ValidateLimit limit<=0 用默认值；超过 max → INVALID_ARGUMENT。
func ValidateLimit(limit int32, def, max int32) (int32, error) {
	if limit < 0 {
		return 0, InvalidArgument("limit must be >= 0")
	}
	if limit == 0 {
		return def, nil
	}
	if limit > max {
		return 0, InvalidArgument("limit exceeds maximum")
	}
	return limit, nil
}

// ValidateTimeRange startTime>endTime 或非法时间 → INVALID_ARGUMENT。
func ValidateTimeRange(startTime, endTime int64) error {
	if startTime < 0 || endTime < 0 {
		return InvalidArgument("time must be UnixNano >= 0")
	}
	if startTime > 0 && endTime > 0 && startTime > endTime {
		return InvalidArgument("startTime must be <= endTime")
	}
	return nil
}

// CoverageStatus 取值。
const (
	CoverageComplete = "complete"
	CoveragePartial  = "partial"
	CoverageUnknown  = "unknown"
)

// ResolveCoverage 根据 observation_gap 判定覆盖状态。
// 只要窗口内存在缺口，禁止返回 complete。
func ResolveCoverage(gapDropped uint64, gapFound bool, windowCovered bool) (string, uint64) {
	switch {
	case gapFound && gapDropped > 0:
		return CoveragePartial, gapDropped
	case gapFound:
		// 有 gap 标记但计数解析失败
		return CoverageUnknown, 0
	case !windowCovered:
		return CoverageUnknown, 0
	default:
		return CoverageComplete, 0
	}
}
