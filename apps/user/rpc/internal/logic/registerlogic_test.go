package logic

import (
	"context"
	"testing"

	"github.com/IM_System/apps/user/rpc/user"
)

func TestRegisterLogic_Register(t *testing.T) {
	type args struct {
		in *user.RegisterReq
	}
	tests := []struct {
		name      string // description of this test case
		args      args
		wantPrint bool
		wantErr   bool
	}{
		// TODO: Add test cases.
		{
			"1", args{in: &user.RegisterReq{
				Phone:    "12345678901",
				Nickname: "yyy",
				Password: "123456",
				Avatar:   "png.jpg",
				Sex:      1,
			}}, true, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewRegisterLogic(context.Background(), svcCtx)
			got, gotErr := l.Register(tt.args.in)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Register() failed: %v", gotErr)
				}
				return
			}
			if tt.wantPrint {
				t.Log(tt.name, got)
			}
			if tt.wantErr {
				t.Fatal("Register() succeeded unexpectedly")
			}
			if got == nil {
				t.Errorf("Register() = nil, want non-nil")
			}
		})
	}
}
