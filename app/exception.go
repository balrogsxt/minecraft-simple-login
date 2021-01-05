package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strings"
)

type YggdrasilException struct {
	Error        string
	ErrorMessage string
	Code         int
}

func ThrowYggdrasilException(error interface{}, errorMsg interface{}, code ...int) {
	c := 403
	if len(code) >= 1 {
		c = code[0]
	}

	panic(YggdrasilException{
		Code:         c,
		Error:        fmt.Sprintf("%v", error),
		ErrorMessage: strings.ReplaceAll(fmt.Sprintf("%v", errorMsg), "&", "§"),
	})
}

type HttpException struct {
	Status int
	Msg    string
}

func ThrowHttpException(msg string, status ...int) {
	s := 1
	if len(status) >= 1 {
		s = status[0]
	}
	panic(HttpException{
		Msg:    msg,
		Status: s,
	})
}
func BuildHttpResponse(msg string, data ...interface{}) gin.H {
	var d interface{} = nil
	if len(data) > 0 {
		d = data[0]
	}
	return gin.H{
		"status": 0,
		"msg":    msg,
		"data":   d,
	}
}
