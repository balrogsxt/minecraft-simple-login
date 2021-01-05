package app

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

var engine *xorm.Engine

func InitDatabase(config *Config) error {
	conf := config.Mysql

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8", conf.User, conf.Password, conf.Host, conf.Port, conf.Name)
	engine, _ = xorm.NewEngine("mysql", dsn)
	err := engine.Ping()
	if err != nil {
		return err
	}
	return nil
}

func GetDb() *xorm.Engine {
	return engine
}

//离线玩家数据表
type Player struct {
	Id       int64
	Uuid     string `xorm:"uuid" json:"uuid"`
	Email    string //登录邮箱
	Password string //登录密码
	Name     string //昵称
	Skin     string //皮肤地址
	Time     int64  //注册时间
	Sign     string //用于账号修改密码后使accessToken失效的策略
}
