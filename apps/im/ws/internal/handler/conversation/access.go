package conversation

import (
	"context"
	"time"

	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/useraccess"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const permissionLookupTimeout = 3 * time.Second

func authorizeChat(ctx context.Context, services *svc.ServiceContext, uid, receiver, supplied string, chatType constants.ChatType) (string, error) {
	id, err := useraccess.ResolveConversation(uid, receiver, supplied, chatType)
	if err != nil {
		return "", err
	}
	if chatType == constants.GroupChatType {
		err = useraccess.RequireGroup(ctx, func(ctx context.Context, groupID string) ([]useraccess.GroupMember, error) {
			resp, err := services.Social.Groupusers(ctx, &social.GroupusersReq{GroupId: groupID})
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
		}, uid, receiver, false)
	}
	return id, err
}

// Validate the whole batch before producing Kafka records; knowing a message
// ID must not allow altering read state in a different conversation.
func authorizeRead(ctx context.Context, services *svc.ServiceContext, uid, receiver, supplied string, chatType constants.ChatType, ids []string) (string, error) {
	id, err := authorizeChat(ctx, services, uid, receiver, supplied, chatType)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 || len(ids) > 100 {
		return "", status.Error(codes.InvalidArgument, "已读消息数量必须为 1 至 100")
	}
	unique := make(map[string]struct{}, len(ids))
	for _, msgID := range ids {
		if _, err := bson.ObjectIDFromHex(msgID); err != nil {
			return "", status.Error(codes.InvalidArgument, "消息 ID 格式无效")
		}
		unique[msgID] = struct{}{}
	}
	rows, err := services.ChatLogModel.ListByIds(ctx, ids)
	if err != nil {
		return "", err
	}
	if len(rows) != len(unique) {
		return "", useraccess.Denied()
	}
	for _, row := range rows {
		if row == nil || row.ConversationId != id || row.ChatType != chatType {
			return "", useraccess.Denied()
		}
		if chatType == constants.SingleChatType && row.RecvId != uid {
			return "", useraccess.Denied() // Only the recipient can acknowledge reading.
		}
	}
	return id, nil
}
