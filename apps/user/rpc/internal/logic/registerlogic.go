package logic

import (
	"context"
	"database/sql"
	"time"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/IM_System/pkg/encrypt"
	"github.com/IM_System/pkg/wuid"
	"github.com/IM_System/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrPhoneIsRegister = xerr.New(xerr.SERVER_COMMON_ERR, "手机号已注册")
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *user.RegisterReq) (*user.RegisterResp, error) {
	// todo: add your logic here and delete this line

	// 根据手机号查询用户是否存在
	userEntity, err := l.svcCtx.UsersModel.FindPhone(l.ctx, in.Phone)
	if err != nil && err != models.ErrNotFound {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find user by phone error: %v, req: %v", err, in.Phone)
	}

	if userEntity != nil {
		return nil, errors.WithStack(ErrPhoneIsRegister)
	}

	// 定义用户数据
	userEntity = &models.Users{
		Id:       wuid.GenUid(l.svcCtx.Config.Mysql.DataSource),
		Avatar:   in.Avatar,
		Nickname: in.Nickname,
		Phone:    in.Phone,
		Sex: sql.NullInt64{
			Int64: int64(in.Sex),
			Valid: true,
		},
	}

	if len(in.Password) > 0 {
		genpassword, err := encrypt.GenPasswordHash([]byte(in.Password))
		if err != nil {
			return nil, errors.Wrapf(xerr.NewDBErr(), "generate password hash error: %v, req: %v", err, in)
		}
		userEntity.Password = sql.NullString{
			String: string(genpassword),
			Valid:  true,
		}
	}

	// 插入用户数据
	_, err = l.svcCtx.UsersModel.Insert(l.ctx, userEntity)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "insert user error: %v, req: %v", err, in)
	}

	// 生成token
	now := time.Now().Unix()
	token, err := ctxdata.GetTokenKey(l.svcCtx.Config.Jwt.AccessSecret, now, l.svcCtx.Config.Jwt.AccessExpire, userEntity.Id)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "ctxdata get jwt token err: %v", err)
	}

	return &user.RegisterResp{
		Token:  token,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
