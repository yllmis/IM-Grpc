package logic

import (
	"context"
	"time"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/IM_System/pkg/encrypt"
	"github.com/IM_System/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrPhoneNotRegister = xerr.New(xerr.SERVER_COMMON_ERR, "手机号未注册")
	ErrPasswordError    = xerr.New(xerr.SERVER_COMMON_ERR, "密码错误")
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *user.LoginReq) (*user.LoginResp, error) {
	// todo: add your logic here and delete this line

	// 根据手机号查询用户是否存在
	userEntity, err := l.svcCtx.UsersModel.FindPhone(l.ctx, in.Phone)
	if err != nil {
		if err == models.ErrNotFound {
			return nil, errors.WithStack(ErrPhoneNotRegister)
		}
		return nil, errors.Wrapf(xerr.NewDBErr(), "find user by phone error: %v, req: %v", err, in.Phone)
	}

	// 验证密码
	if !encrypt.ValidatePasswordHash(in.Password, userEntity.Password.String) {
		return nil, errors.WithStack(ErrPasswordError)
	}

	// 生成token
	now := time.Now().Unix()
	token, err := ctxdata.GetTokenKey(l.svcCtx.Config.Jwt.AccessSecret, now, l.svcCtx.Config.Jwt.AccessExpire, userEntity.Id)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "ctxdata get jwt token err: %v", err)
	}

	return &user.LoginResp{
		Token:  token,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
