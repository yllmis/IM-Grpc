// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package friend

import (
	"context"

	"github.com/IM_System/apps/social/api/internal/svc"
	"github.com/IM_System/apps/social/api/internal/types"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type FriendListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 好友列表
func NewFriendListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FriendListLogic {
	return &FriendListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FriendListLogic) FriendList(req *types.FriendListReq) (resp *types.FriendListResp, err error) {
	// todo: add your logic here and delete this line

	uid := ctxdata.GetUid(l.ctx)

	friends, err := l.svcCtx.SocialRpc.FriendList(l.ctx, &social.FriendListReq{
		UserId: uid,
	})

	if err != nil {
		return nil, err
	}

	if len(friends.Friends) == 0 {
		return
	}

	// 获取好友列表的用户id
	uids := make([]string, 0, len(friends.Friends))
	for _, friend := range friends.Friends {
		uids = append(uids, friend.FriendUid)
	}

	// 根据ids获取好友信息
	friendInfo, err := l.svcCtx.UserRpc.FindUser(l.ctx, &user.FindUserReq{
		Ids: uids,
	})
	if err != nil {
		return &types.FriendListResp{}, err
	}

	userRecords := make(map[string]*user.UserEntity)
	for i, _ := range friendInfo.Users {
		userRecords[friendInfo.Users[i].Id] = friendInfo.Users[i]
	}

	respList := make([]*types.Friends, 0, len(friends.Friends))

	for _, v := range friends.Friends {
		friend := &types.Friends{
			Id:        v.Id,
			FriendUid: v.FriendUid,
		}
		if user, ok := userRecords[v.FriendUid]; ok {
			friend.Nickname = user.Nickname
			friend.Avatar = user.Avatar
		}
		respList = append(respList, friend)
	}

	return &types.FriendListResp{
		List: respList,
	}, nil
}
