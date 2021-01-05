package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"regexp"
)

var jsonConfig = jsoniter.ConfigCompatibleWithStandardLibrary

//任意结构体转json字符串
func JsonEncode(v interface{}) string {
	s, _ := jsonConfig.Marshal(&v)
	return string(s)
}

//字符串转结构体
func JsonDecode(jsonString string, v interface{}) error {
	return jsonConfig.Unmarshal([]byte(jsonString), &v)
}

func Md5String(str interface{}) string {
	m := md5.New()
	m.Write([]byte(fmt.Sprintf("%v", str)))
	return hex.EncodeToString(m.Sum(nil))
}

func IsEmail(str string) bool {
	pattern := "[\\w!#$%&'*+/=?^_`{|}~-]+(?:\\.[\\w!#$%&'*+/=?^_`{|}~-]+)*@(?:[\\w](?:[\\w-]*[\\w])?\\.)+[\\w](?:[\\w-]*[\\w])?"
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(str)
}
