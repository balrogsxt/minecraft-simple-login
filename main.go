package main

import (
	"fmt"
	"github.com/balrogsxt/minecraft-login/app"
	"github.com/balrogsxt/minecraft-login/controller"
	"github.com/balrogsxt/minecraft-login/utils/logger"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			logger.Fatal("运行错误: %#v", err)
		}
	}()

	config := app.GetConfig()
	if err := app.InitDatabase(config); err != nil {
		logger.Fatal("数据库加载失败: %s", err.Error())
	} else {
		logger.Info("数据库连接成功")
	}
	if err := app.InitRedis(config); err != nil {
		logger.Fatal("Redis加载失败: %s", err.Error())
	} else {
		logger.Info("Redis连接成功")
	}

	r := gin.New()
	gin.SetMode(gin.ReleaseMode)
	//r := gin.Default()
	r.Use(tryCatch())
	r.Use(CorsAllow())
	//离线玩家注册、设置、皮肤等
	r.POST("/login", controller.OffineRegisterOrLogin)
	auth := r.Group("/player", controller.AuthMiddleware)
	{
		auth.GET("/info", controller.OffineInfo)              //基础信息
		auth.POST("/uploadSkin", controller.OffineUploadSkin) //上传玩家皮肤
		auth.POST("/setName", controller.OffineSetPlayerName) //设置玩家名称
		auth.POST("/editpwd", controller.OffineEditPassword)  //修改密码
		auth.GET("/logout", controller.OffineLogout)          //退出登录
	}
	//yggdrasil API
	r.GET("/", controller.Index)
	r.POST("/authserver/validate", controller.Validate)
	r.POST("/authserver/refresh", controller.Refresh)
	r.POST("/authserver/authenticate", controller.Login)
	r.POST("/sessionserver/session/minecraft/join", controller.Join)
	r.GET("/sessionserver/session/minecraft/hasJoined", controller.HasJoined)
	r.GET("/sessionserver/session/minecraft/profile/:uuid", controller.GetSkin)

	r.Run(fmt.Sprintf("0.0.0.0:%v", config.HttpPort))
}
func CorsAllow() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
	}
}
func tryCatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if ex, flag := r.(app.YggdrasilException); flag {
					c.JSON(ex.Code, gin.H{
						"error":        ex.Error,
						"errorMessage": ex.ErrorMessage,
					})
				} else if ex, flag := r.(app.HttpException); flag {
					c.JSON(200, gin.H{
						"status": ex.Status,
						"msg":    ex.Msg,
					})
				} else {
					log.Println("未知的错误信息:", r)
					c.JSON(200, gin.H{
						"error":        "系统错误",
						"errorMessage": "未知的错误信息!",
					})
				}
			}
		}()
		c.Next()
	}
}
