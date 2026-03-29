package xerr

var codeText = map[int]string{
	SERVER_COMMON_ERR: "服务器异常,请稍后再试",
	REQUEST_PARAM_ERR: "请求参数错误",
	DB_ERROR:          "数据库繁忙,请稍后再试",
}

func ErrMsg(errcode int) string {
	if msg, ok := codeText[errcode]; ok {
		return msg
	}
	return codeText[SERVER_COMMON_ERR]

}
