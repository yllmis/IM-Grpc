package resultx

import (
	"context"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthorizationHTTPStatus(t *testing.T) {
	for _, tc := range []struct {
		code codes.Code
		want int
	}{
		{codes.Unauthenticated, http.StatusUnauthorized}, {codes.PermissionDenied, http.StatusForbidden}, {codes.InvalidArgument, http.StatusBadRequest},
	} {
		got, _ := ErrHandler("test")(context.Background(), status.Error(tc.code, "test error"))
		if got != tc.want {
			t.Fatalf("code=%s http=%d", tc.code, got)
		}
	}
}
