package logic

import (
	"context"

	"github.com/IM_System/apps/im/api/internal/svc"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/useraccess"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func requireGroup(ctx context.Context, services *svc.ServiceContext, uid, groupID string) error {
	return useraccess.RequireGroup(ctx, func(ctx context.Context, id string) ([]useraccess.GroupMember, error) {
		resp, err := services.Social.Groupusers(ctx, &social.GroupusersReq{GroupId: id})
		if err != nil {
			return nil, err
		}
		var members []useraccess.GroupMember
		if resp != nil {
			for _, m := range resp.List {
				if m != nil {
					members = append(members, useraccess.GroupMember{UserID: m.UserId, Role: constants.GroupRoleLevel(m.RoleLevel)})
				}
			}
		}
		return members, nil
	}, uid, groupID, false)
}

func requireConversation(ctx context.Context, services *svc.ServiceContext, uid, id string) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "会话 ID 不能为空")
	}
	resp, err := services.Im.GetConversations(ctx, &im.GetConversationsReq{UserId: uid})
	if err != nil {
		return err
	}
	if resp == nil {
		return useraccess.Denied()
	}
	conversation := resp.ConversationList[id]
	if conversation == nil || conversation.ConversationId != id {
		return useraccess.Denied()
	}
	switch constants.ChatType(conversation.ChatType) {
	case constants.SingleChatType:
		return nil
	case constants.GroupChatType:
		return requireGroup(ctx, services, uid, id)
	default:
		return useraccess.Denied()
	}
}
