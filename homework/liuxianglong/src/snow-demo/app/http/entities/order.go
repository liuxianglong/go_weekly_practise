package entities



/*
 * validator.v9文档
 * 地址https://godoc.org/gopkg.in/go-playground/validator.v9
 * 列了几个大家可能会用到的，如有遗漏，请看上面文档
 */

//请求数据结构

type OrderValidatorRequest struct {
	OrderNo string `json:"orderNo" example:"snow"`
}
