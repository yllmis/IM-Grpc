package ctxdata

import "github.com/golang-jwt/jwt"

const Identify = "yllmis-im"

// GetTokenKey 获取token key(密钥,签发时间,过期时间,用户id)
func GetTokenKey(secretKey string, iat, second int64, uid string) (string, error) {
	clamis := make(jwt.MapClaims)
	clamis["iat"] = iat // 签发时间
	clamis["exp"] = iat + second
	clamis[Identify] = uid // 用户id

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, clamis)
	token.Claims = clamis

	return token.SignedString([]byte(secretKey))
}
