package logic

import (
	"context"
	"errors"
	"time"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/IM_System/pkg/encrypt"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrPhoneNotRegister = errors.New("手机号未注册")
	ErrPasswordError    = errors.New("密码错误")
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
			return nil, ErrPhoneNotRegister
		}
		return nil, err
	}

	// 验证密码
	if !encrypt.ValidatePasswordHash(in.Password, userEntity.Password.String) {
		return nil, ErrPasswordError
	}

	// 生成token
	now := time.Now().Unix()
	token, err := ctxdata.GetTokenKey(l.svcCtx.Config.Jwt.AccessSecret, now, l.svcCtx.Config.Jwt.AccessExpire, userEntity.Id)
	if err != nil {
		return nil, err
	}

	return &user.LoginResp{
		Token:  token,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
