// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/constants"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupOnlineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 群在线用户
func NewGroupOnlineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupOnlineLogic {
	return &GroupOnlineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupOnlineLogic) GroupOnline(req *types.GroupOnlineReq) (resp *types.GroupOnlineResp, err error) {
	// todo: add your logic here and delete this line

	groupUsers, err := l.svcCtx.SocialRpc.Groupusers(l.ctx, &social.GroupusersReq{
		GroupId: req.GroupId,
	})

	if err != nil || len(groupUsers.List) == 0 {
		return &types.GroupOnlineResp{}, nil
	}

	// 查询，缓存中查询在线的用户
	uids := make([]string, 0, len(groupUsers.List))
	for _, groupmember := range groupUsers.List {
		uids = append(uids, groupmember.UserId)
	}

	onlines, err := l.svcCtx.Redis.Hgetall(constants.REDIS_ONLINE_USERS)
	if err != nil {
		return nil, err
	}

	respOnlineList := make(map[string]bool, len(uids))
	for _, uid := range uids {
		if _, ok := onlines[uid]; ok {
			respOnlineList[uid] = true
		} else {
			respOnlineList[uid] = false
		}
	}
	return &types.GroupOnlineResp{
		OnlineList: respOnlineList,
	}, nil
}
