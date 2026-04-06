package handler

import (
	"context"
	"net/http"

	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/token"
)

type JwtAuth struct {
	srv    *svc.ServiceContext
	parser *token.TokenParser
	logx.Logger
}

func NewJwtAuth(srv *svc.ServiceContext) *JwtAuth {
	return &JwtAuth{
		srv:    srv,
		parser: token.NewTokenParser(),
		Logger: logx.WithContext(context.Background()),
	}
}

func (j *JwtAuth) Auth(w http.ResponseWriter, r *http.Request) bool {
	tok, err := j.parser.ParseToken(r, j.srv.Config.JwtAuth.AccessSecret, "")
	if err != nil {
		j.Errorf("Failed to parse token: %v", err)
		return false
	}

	if !tok.Valid {
		return false
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	*r = *r.WithContext(context.WithValue(r.Context(), ctxdata.Identify, claims[ctxdata.Identify]))

	return true
}

func (j *JwtAuth) UserId(r *http.Request) string {
	return ctxdata.GetUid(r.Context())
}
