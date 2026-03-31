package logic

import (
	"path/filepath"

	"github.com/IM_System/apps/user/rpc/internal/config"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
)

var svcCtx *svc.ServiceContext

func init() {
	var c config.Config
	conf.MustLoad(filepath.Join("../../etc/dev/user copy.yaml"), &c)

	svcCtx = svc.NewServiceContext(c)
}
