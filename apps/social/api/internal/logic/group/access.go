package group

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/useraccess"
)

func requireGroupAccess(ctx context.Context, services *svc.ServiceContext, id string, manager bool) error {
	uid, err := useraccess.RequireUser(ctx)
	if err != nil {
		return err
	}
	return useraccess.RequireGroup(ctx, func(ctx context.Context, groupID string) ([]useraccess.GroupMember, error) {
		resp, err := services.SocialRpc.Groupusers(ctx, &social.GroupusersReq{GroupId: groupID})
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
	}, uid, id, manager)
}
