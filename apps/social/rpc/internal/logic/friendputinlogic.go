package logic

import (
	"context"
	"database/sql"
	"time"

	"github.com/IM_System/apps/social/rpc/internal/svc"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/social/socialmodels"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type FriendPutInLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFriendPutInLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FriendPutInLogic {
	return &FriendPutInLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FriendPutInLogic) FriendPutIn(in *social.FriendPutInReq) (*social.FriendPutInResp, error) {
	// todo: add your logic here and delete this line

	// 申请人与被申请人是否为好友关系
	friends, err := l.svcCtx.FriendsModel.FindOneByUserIdFriendUid(l.ctx, in.UserId, in.Requid)
	if err != nil && err != socialmodels.ErrNotFound {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find friend by uid and fid err %v req %v", err, in)
	}
	if friends != nil {
		return &social.FriendPutInResp{}, nil
	}

	// 是否有过申请，不成功，未处理
	friendReqs, err := l.svcCtx.FriendRequestsModel.FindOneByUidAndUerId(l.ctx, in.Requid, in.UserId)
	if err != nil && err != socialmodels.ErrNotFound {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find friendRequest by uid and fid err %v req %v", err, in)
	}
	if friendReqs != nil {
		return &social.FriendPutInResp{}, nil
	}

	// 申请人可以申请, 创建申请记录
	_, err = l.svcCtx.FriendRequestsModel.Insert(l.ctx, &socialmodels.FriendRequests{
		UserId: in.UserId,
		ReqUid: in.Requid,
		ReqMsg: sql.NullString{
			Valid:  true,
			String: in.ReqMsg,
		},
		ReqTime: time.Unix(in.ReqTime, 0),
		HandleResult: sql.NullInt64{
			Int64: int64(constants.NoHandlerResult),
			Valid: true,
		},
	})

	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "insert friendRequest err %v req %v", err, in)
	}

	return &social.FriendPutInResp{}, nil
}
