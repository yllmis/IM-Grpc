// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/user/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupUserListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 成员列表列表
func NewGroupUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupUserListLogic {
	return &GroupUserListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupUserListLogic) GroupUserList(req *types.GroupUserListReq) (resp *types.GroupUserListResp, err error) {
	// todo: add your logic here and delete this line

	groupUsers, err := l.svcCtx.SocialRpc.Groupusers(l.ctx, &social.GroupusersReq{
		GroupId: req.GroupId,
	})
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	uids := make([]string, 0, len(groupUsers.List))
	for _, v := range groupUsers.List {
		uids = append(uids, v.UserId)
	}

	userList, err := l.svcCtx.UserRpc.FindUser(l.ctx, &userclient.FindUserReq{
		Ids: uids,
	})
	if err != nil {
		return nil, err
	}

	// 防止查出来张冠李戴的情况，mysql中in查询的结果是无序的，所以我们需要把查出来的用户信息进行一个记录，记录成一个map
	userRecords := make(map[string]*userclient.UserEntity, len(userList.Users))
	for i, _ := range userList.Users {
		userRecords[userList.Users[i].Id] = userList.Users[i]
	}

	respList := make([]*types.GroupMembers, 0, len(groupUsers.List))
	for _, v := range groupUsers.List {

		member := &types.GroupMembers{
			Id:        int64(v.Id),
			GroupId:   v.GroupId,
			UserId:    v.UserId,
			RoleLevel: int(v.RoleLevel),
		}
		// 根据对应的用户id去查找对应的用户信息
		if u, ok := userRecords[v.UserId]; ok {
			member.Nickname = u.Nickname
			member.UserAvatarUrl = u.Avatar
		}
		respList = append(respList, member)
	}
	return &types.GroupUserListResp{List: respList}, nil
}
