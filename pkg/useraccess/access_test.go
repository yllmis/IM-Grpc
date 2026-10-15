package useraccess

import (
	"context"
	"errors"
	"testing"

	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequireUser(t *testing.T) {
	for _, uid := range []string{"", " 1", "1"} {
		ctx := context.WithValue(context.Background(), ctxdata.Identify, uid)
		got, err := RequireUser(ctx)
		if uid == "1" {
			if err != nil || got != uid {
				t.Fatal(err)
			}
		} else if status.Code(err) != codes.Unauthenticated {
			t.Fatal("missing identity accepted")
		}
	}
}

func TestInternalWSRequiresAuthenticatedSystemIdentity(t *testing.T) {
	for _, uid := range []string{"", "1", constants.SYSTEM_ROOT_ID} {
		err := RequireSystem(uid)
		if uid == constants.SYSTEM_ROOT_ID {
			if err != nil {
				t.Fatal(err)
			}
		} else if status.Code(err) != codes.PermissionDenied {
			t.Fatal("ordinary user can push internally")
		}
	}
}

func TestResolveConversation(t *testing.T) {
	for _, tc := range []struct {
		uid, recv, supplied string
		kind                constants.ChatType
		want                string
		code                codes.Code
	}{
		{"1", "2", "", constants.SingleChatType, "1_2", codes.OK},
		{"2", "1", "1_2", constants.SingleChatType, "1_2", codes.OK},
		{"1", "2", "3_4", constants.SingleChatType, "", codes.PermissionDenied},
		{"1", "g1", "", constants.GroupChatType, "g1", codes.OK},
		{"1", "g1", "g2", constants.GroupChatType, "", codes.PermissionDenied},
		{"", "2", "", constants.SingleChatType, "", codes.Unauthenticated},
		{"1", "", "", constants.SingleChatType, "", codes.InvalidArgument},
		{"1", "2", "", constants.ChatType(99), "", codes.InvalidArgument},
	} {
		got, err := ResolveConversation(tc.uid, tc.recv, tc.supplied, tc.kind)
		if got != tc.want || status.Code(err) != tc.code {
			t.Fatalf("%+v got=%s err=%v", tc, got, err)
		}
	}
}

func TestGroupMemberAndManagerPermissions(t *testing.T) {
	lookup := func(context.Context, string) ([]GroupMember, error) {
		return []GroupMember{{"owner", constants.CreatorGroupRoleLevel}, {"admin", constants.ManagerGroupRoleLevel}, {"member", constants.AtLargeGroupRoleLevel}}, nil
	}
	for _, uid := range []string{"owner", "admin", "member", "outsider"} {
		for _, manager := range []bool{false, true} {
			err := RequireGroup(context.Background(), lookup, uid, "g1", manager)
			allowed := uid != "outsider" && (!manager || uid != "member")
			if (err == nil) != allowed {
				t.Fatalf("uid=%s manager=%v err=%v", uid, manager, err)
			}
		}
	}
	storageError := errors.New("storage unavailable")
	err := RequireGroup(context.Background(), func(context.Context, string) ([]GroupMember, error) { return nil, storageError }, "owner", "g1", false)
	if !errors.Is(err, storageError) {
		t.Fatal("failed membership lookup became access grant")
	}
}
