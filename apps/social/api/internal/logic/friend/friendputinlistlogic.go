// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package friend

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/user/rpc/userclient"
	"github.com/IM_System/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type FriendPutInListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 好友申请列表
func NewFriendPutInListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FriendPutInListLogic {
	return &FriendPutInListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FriendPutInListLogic) FriendPutInList(req *types.FriendPutInListReq) (resp *types.FriendPutInListResp, err error) {

	uid := ctxdata.GetUid(l.ctx)
	friendPutList, err := l.svcCtx.SocialRpc.FriendPutInList(l.ctx, &social.FriendPutInListReq{
		UserId: uid,
	})
	if err != nil {
		return nil, err
	}

	if friendPutList == nil || len(friendPutList.FriendReqs) == 0 {
		return
	}

	uids := make([]string, 0, len(friendPutList.FriendReqs))
	var friendReqs []*types.FriendRequests
	for _, friendReq := range friendPutList.FriendReqs {
		friendReqs = append(friendReqs, &types.FriendRequests{
			UserId:       uid,
			ReqUid:       friendReq.ReqUid,
			ReqMsg:       friendReq.ReqMsg,
			ReqTime:      friendReq.ReqTime,
			HandleResult: int(friendReq.HandleResult),
		})
		uids = append(uids, friendReq.ReqUid)
	}

	// 获取用户信息
	userRecords := make(map[string]*userclient.UserEntity)
	userList, err := l.svcCtx.UserRpc.FindUser(l.ctx, &userclient.FindUserReq{Ids: uids})
	if err != nil {
		logx.Errorf("userRpc.FindUser failed: %v", err)
	} else {
		for i := range userList.Users {
			userRecords[userList.Users[i].Id] = userList.Users[i]
		}
	}

	respList := make([]*types.FriendRequests, 0, len(friendReqs))
	for _, v := range friendReqs {
		if u, ok := userRecords[v.ReqUid]; ok {
			v.Nickname = u.Nickname
			v.Avatar = u.Avatar
		}
		respList = append(respList, v)
	}

	resp = &types.FriendPutInListResp{
		List: respList,
	}

	return
}
