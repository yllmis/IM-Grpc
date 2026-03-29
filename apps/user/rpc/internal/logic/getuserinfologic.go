package logic

import (
	"context"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

var ErrUserNotFound = xerr.New(xerr.SERVER_COMMON_ERR, "没有找到该用户")

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserInfoLogic) GetUserInfo(in *user.GetUserInfoReq) (*user.GetUserInfoResp, error) {
	// todo: add your logic here and delete this line

	userEntity, err := l.svcCtx.UsersModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == models.ErrNotFound {
			return nil, errors.WithStack(ErrUserNotFound)
		}
		return nil, errors.Wrapf(xerr.NewDBErr(), "find user by id error: %v, req: %v", err, in.Id)
	}

	var resp user.UserEntity
	copier.Copy(&resp, userEntity)

	return &user.GetUserInfoResp{
		User: &resp,
	}, nil
}
