// Package useraccess enforces IM end-user object permissions. Service
// credentials never grant an end user access to somebody else's conversation.
package useraccess

import (
	"context"
	"strings"

	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/IM_System/pkg/wuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RequireUser(ctx context.Context) (string, error) {
	uid := ctxdata.GetUid(ctx)
	if uid == "" || strings.TrimSpace(uid) != uid {
		return "", status.Error(codes.Unauthenticated, "需要有效的用户身份")
	}
	return uid, nil
}

func Denied() error { return status.Error(codes.PermissionDenied, "无权访问该资源") }

// The UID must come from a verified WS connection, never Message.FormId.
func RequireSystem(uid string) error {
	if uid != constants.SYSTEM_ROOT_ID {
		return Denied()
	}
	return nil
}

type GroupMember struct {
	UserID string
	Role   constants.GroupRoleLevel
}

type GroupLookup func(context.Context, string) ([]GroupMember, error)

func RequireGroup(ctx context.Context, lookup GroupLookup, uid, groupID string, manager bool) error {
	if uid == "" {
		return status.Error(codes.Unauthenticated, "需要有效的用户身份")
	}
	if groupID == "" {
		return status.Error(codes.InvalidArgument, "群 ID 不能为空")
	}
	members, err := lookup(ctx, groupID)
	if err != nil {
		return err // A failed lookup must never grant access.
	}
	for _, m := range members {
		if m.UserID == uid {
			if !manager || m.Role == constants.CreatorGroupRoleLevel || m.Role == constants.ManagerGroupRoleLevel {
				return nil
			}
			break
		}
	}
	return Denied()
}

// ResolveConversation rejects a caller-supplied conversation ID that does not
// match its actual participants. Private messaging itself remains unrestricted
// by friendship, matching the existing product's behavior.
func ResolveConversation(uid, receiver, supplied string, chatType constants.ChatType) (string, error) {
	if uid == "" {
		return "", status.Error(codes.Unauthenticated, "需要有效的用户身份")
	}
	if receiver == "" {
		return "", status.Error(codes.InvalidArgument, "接收方不能为空")
	}
	var canonical string
	switch chatType {
	case constants.SingleChatType:
		canonical = wuid.CombineId(uid, receiver)
	case constants.GroupChatType:
		canonical = receiver
	default:
		return "", status.Error(codes.InvalidArgument, "不支持的聊天类型")
	}
	if supplied != "" && supplied != canonical {
		return "", Denied()
	}
	return canonical, nil
}
