package readquery

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestErrorSemanticsAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want codes.Code
	}{
		{context.DeadlineExceeded, codes.DeadlineExceeded}, {context.Canceled, codes.Canceled},
		{errors.New("connection refused"), codes.Unavailable}, {errors.New("secret mysql url"), codes.Internal},
		{status.Error(codes.PermissionDenied, "denied"), codes.PermissionDenied},
		{fmt.Errorf("wrapped: %w", status.Error(codes.Unimplemented, "unsupported")), codes.Unimplemented},
	} {
		got := MapError(tc.err)
		if status.Code(got) != tc.want || strings.Contains(got.Error(), "secret mysql url") {
			t.Fatalf("%v -> %v", tc.err, got)
		}
	}
}
