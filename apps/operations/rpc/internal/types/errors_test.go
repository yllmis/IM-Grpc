package types

import (
	"context"
	"errors"
	"testing"

	"github.com/IM_System/apps/operations/rpc/operationsmodels"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func codeOf(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return status.Code(err)
}

func TestValidateMessageID(t *testing.T) {
	if codeOf(ValidateMessageID("")) != codes.InvalidArgument {
		t.Fatal("empty messageId must be INVALID_ARGUMENT")
	}
	if codeOf(ValidateMessageID("not-hex")) != codes.InvalidArgument {
		t.Fatal("invalid hex must be INVALID_ARGUMENT")
	}
	if codeOf(ValidateMessageID("665f1c0000000000000000aa")) != codes.OK {
		t.Fatal("valid 24hex must pass")
	}
}

func TestValidateLimit(t *testing.T) {
	// limit<=0 → default
	n, err := ValidateLimit(0, 50, 200)
	if err != nil || n != 50 {
		t.Fatalf("default limit: %d %v", n, err)
	}
	n, err = ValidateLimit(10, 50, 200)
	if err != nil || n != 10 {
		t.Fatalf("normal limit: %d %v", n, err)
	}
	_, err = ValidateLimit(201, 50, 200)
	if codeOf(err) != codes.InvalidArgument {
		t.Fatal("over max must be INVALID_ARGUMENT")
	}
	_, err = ValidateLimit(-1, 50, 200)
	if codeOf(err) != codes.InvalidArgument {
		t.Fatal("negative limit must be INVALID_ARGUMENT")
	}
}

func TestValidateTimeRange(t *testing.T) {
	if codeOf(ValidateTimeRange(10, 5)) != codes.InvalidArgument {
		t.Fatal("start>end must be INVALID_ARGUMENT")
	}
	if codeOf(ValidateTimeRange(-1, 0)) != codes.InvalidArgument {
		t.Fatal("negative time must be INVALID_ARGUMENT")
	}
	if codeOf(ValidateTimeRange(5, 10)) != codes.OK {
		t.Fatal("valid range must pass")
	}
	if codeOf(ValidateTimeRange(0, 0)) != codes.OK {
		t.Fatal("zero range means unlimited")
	}
}

func TestMapQueryError(t *testing.T) {
	// 超时不得被当成“无数据”
	if codeOf(MapQueryError(context.DeadlineExceeded)) != codes.DeadlineExceeded {
		t.Fatal("deadline must map to DEADLINE_EXCEEDED")
	}
	if codeOf(MapQueryError(errors.New("context deadline exceeded"))) != codes.DeadlineExceeded {
		t.Fatal("wrapped deadline must map to DEADLINE_EXCEEDED")
	}
	if codeOf(MapQueryError(errors.New("connection refused"))) != codes.Unavailable {
		t.Fatal("connection refused must map to UNAVAILABLE")
	}
	if codeOf(MapQueryError(errors.New("boom"))) != codes.Internal {
		t.Fatal("unknown must map to INTERNAL")
	}
	// ErrNotFound 不是错误（调用方转空结果）
	if MapQueryError(operationsmodels.ErrNotFound) != nil {
		t.Fatal("not found must not be a grpc error")
	}
}

func TestResolveCoverage(t *testing.T) {
	st, d := ResolveCoverage(3, true, true)
	if st != CoveragePartial || d != 3 {
		t.Fatalf("gap => partial: %s %d", st, d)
	}
	st, _ = ResolveCoverage(0, true, true)
	if st != CoverageUnknown {
		t.Fatalf("gap without count => unknown, got %s", st)
	}
	st, _ = ResolveCoverage(0, false, false)
	if st != CoverageUnknown {
		t.Fatalf("uncovered window => unknown, got %s", st)
	}
	st, _ = ResolveCoverage(0, false, true)
	if st != CoverageComplete {
		t.Fatalf("no gap + covered => complete, got %s", st)
	}
}

func TestInvalidArgumentCode(t *testing.T) {
	if codeOf(InvalidArgument("x")) != codes.InvalidArgument {
		t.Fatal("InvalidArgument")
	}
	if codeOf(PermissionDenied("x")) != codes.PermissionDenied {
		t.Fatal("PermissionDenied")
	}
	if codeOf(Unimplemented("x")) != codes.Unimplemented {
		t.Fatal("Unimplemented")
	}
	if codeOf(Internal("x")) != codes.Internal {
		t.Fatal("Internal")
	}
}
