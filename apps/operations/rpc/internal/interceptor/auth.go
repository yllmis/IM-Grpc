package interceptor

import "github.com/IM_System/pkg/serviceauth"

// 观测服务复用共享服务身份认证，和领域查询使用相同的 metadata 格式。
const MetadataServiceToken = serviceauth.MetadataServiceToken

type Auth = serviceauth.Auth

var NewAuth = serviceauth.NewAuth
