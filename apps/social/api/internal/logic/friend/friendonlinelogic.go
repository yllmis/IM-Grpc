// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package friend

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type FriendOnlineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 好友在线情况
func NewFriendOnlineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FriendOnlineLogic {
	return &FriendOnlineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FriendOnlineLogic) FriendOnline(req *types.FriendOnlineReq) (resp *types.FriendOnlineResp, err error) {
	// todo: add your logic here and delete this line

	uid := ctxdata.GetUid(l.ctx)

	friendList, err := l.svcCtx.SocialRpc.FriendList(l.ctx, &social.FriendListReq{
		UserId: uid,
	})
	if err != nil || len(friendList.Friends) == 0 {
		return &types.FriendOnlineResp{}, nil
	}

	// 查询，缓存中查询在线的用户
	uids := make([]string, 0, len(friendList.Friends))
	for _, friend := range friendList.Friends {
		uids = append(uids, friend.FriendUid)
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
	return &types.FriendOnlineResp{
		OnlineList: respOnlineList,
	}, nil
}
