package logic

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/IM_System/pkg/encrypt"
	"github.com/IM_System/pkg/wuid"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrPhoneIsRegister = errors.New("手机号已注册")
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
		return nil, err
	}

	if userEntity != nil {
		return nil, ErrPhoneIsRegister
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
			return nil, err
		}
		userEntity.Password = sql.NullString{
			String: string(genpassword),
			Valid:  true,
		}
	}

	// 插入用户数据
	_, err = l.svcCtx.UsersModel.Insert(l.ctx, userEntity)
	if err != nil {
		return nil, err
	}

	// 生成token
	now := time.Now().Unix()
	token, err := ctxdata.GetTokenKey(l.svcCtx.Config.Jwt.AccessSecret, now, l.svcCtx.Config.Jwt.AccessExpire, userEntity.Id)
	if err != nil {
		return nil, err
	}

	return &user.RegisterResp{
		Token:  token,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
