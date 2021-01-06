package controller

import (
	"fmt"
	"github.com/balrogsxt/minecraft-login/api"
	"github.com/balrogsxt/minecraft-login/app"
	"github.com/balrogsxt/minecraft-login/utils"
	"github.com/balrogsxt/minecraft-login/utils/logger"
	"github.com/gin-gonic/gin"
	"os"
	"regexp"
	"strings"
	"time"
)

//登录验证中间件
func AuthMiddleware(c *gin.Context) {
	token := c.GetHeader("authorization")
	if len(token) != 32 {
		c.Abort()
		app.ThrowHttpException("登录验证过期,请重新登录", 403)
	}
	id, err := app.GetRedis().Get(fmt.Sprintf("offine:%s", token))
	if err != nil || len(id) == 0 {
		c.Abort()
		app.ThrowHttpException("登录验证过期,请重新登录账号", 403)
	}
	has, _ := app.GetDb().Where("id = ?", id).Exist()
	if has {
		c.Abort()
		app.ThrowHttpException("登录已失效,请重新登录", 403)
	}
	c.Set("uid", id)
}
func OffineLogout(c *gin.Context) {
	token := c.GetHeader("authorization")
	app.GetRedis().Del(fmt.Sprintf("offine:%s", token))
	c.JSON(200, app.BuildHttpResponse("已退出"))
}
func OffineInfo(c *gin.Context) {
	uid := c.GetString("uid")
	offinePlayer := app.Player{}
	app.GetDb().Where("id = ?", uid).Get(&offinePlayer)
	config := app.GetConfig()

	c.JSON(200, app.BuildHttpResponse("ok", gin.H{
		"id":    uid,
		"email": offinePlayer.Email,
		"skin":  fmt.Sprintf("%s%s", config.SkinUrl, offinePlayer.Skin),
		"uuid":  offinePlayer.Uuid,
		"name":  offinePlayer.Name,
		"time":  time.Unix(offinePlayer.Time, 0).Format("2006-01-02 15:04:05"),
	}))
}

//离线玩家注册、登录
func OffineRegisterOrLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	if !utils.IsEmail(email) || len(password) == 0 {
		app.ThrowHttpException("邮箱格式不正确或密码为空")
	}
	//注册
	session := app.GetDb().NewSession()
	session.Begin()

	offinePlayer := app.Player{}
	has, err := session.Where("email = ?", email).Get(&offinePlayer)
	if err != nil {
		logger.Warning("查询注册账号失败: %s", err.Error())
		app.ThrowHttpException("查询注册账号失败...")
	}
	var t string
	var uid int64
	if has {
		t = "登录"
		//直接登录
		uid = offinePlayer.Id
		if utils.Md5String(password) != offinePlayer.Password {
			app.ThrowHttpException("邮箱或密码不正确")
		}
	} else {
		t = "注册"
		//注册
		player := new(app.Player)
		player.Email = email
		player.Password = utils.Md5String(password)
		player.Time = time.Now().Unix()
		player.Sign = utils.Md5String(fmt.Sprintf("%v%v", time.Now().UnixNano(), uid))

		if _, err := session.InsertOne(player); err != nil {
			logger.Warning("注册用户失败: %s", err.Error())
			app.ThrowHttpException("注册失败,请稍后再试!")
		}
		uid = player.Id
	}

	//随机生成一个token
	token := utils.Md5String(fmt.Sprintf("%v_%v", time.Now().UnixNano(), uid))
	//30天有效
	if err := app.GetRedis().Set(fmt.Sprintf("offine:%s", token), uid, time.Hour*24*30); err != nil {
		session.Rollback()
		logger.Warning(t+"用户redis储存token失败: %s", err.Error())
		app.ThrowHttpException(t + "失败,系统错误")
	}
	session.Commit()

	c.JSON(200, app.BuildHttpResponse(t+"成功", gin.H{
		"token": token,
	}))
}

//设置昵称
func OffineSetPlayerName(c *gin.Context) {
	name := c.PostForm("name")
	if len(name) == 0 || len(name) >= 30 {
		app.ThrowHttpException("游戏昵称过长或不能为空")
	}
	uid := c.GetString("uid")

	uuid := api.GetOffinePlayerUUID(name)
	has, _ := app.GetDb().Where("uuid = ? and id != ?", uuid, uid).Exist()
	if has {
		app.ThrowHttpException("游戏昵称已被绑定,请更换其他昵称!")
	}

	p := new(app.Player)
	p.Name = name
	p.Uuid = api.GetOffinePlayerUUID(name)
	p.Sign = utils.Md5String(fmt.Sprintf("%v%v", time.Now().UnixNano(), uid))

	_, err := app.GetDb().Where("id = ?", uid).Cols("uuid", "name", "sign").Update(p)
	if err != nil {
		logger.Warning("更换游戏昵称失败: %s", err.Error())
		app.ThrowHttpException("更新昵称失败,系统错误")
	}
	c.JSON(200, app.BuildHttpResponse("更新游戏昵称成功"))
}

//上传皮肤
func OffineUploadSkin(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		app.ThrowHttpException("请选择需要上传的皮肤文件!")
	}
	//我就不信啥皮肤有100kb大小
	if file.Size > 1024*100 {
		app.ThrowHttpException("皮肤文件过大,限制100KB!")
	}
	if flag, _ := regexp.MatchString("\\.(png)$", strings.ToLower(file.Filename)); !flag {
		app.ThrowHttpException("皮肤只支持png文件")
	}
	uid := c.GetString("uid")

	offinePlayer := app.Player{}
	app.GetDb().Where("id = ?", uid).Get(&offinePlayer)

	config := app.GetConfig()
	filename := fmt.Sprintf("%s.png", utils.Md5String(fmt.Sprintf("%v_%v", offinePlayer.Uuid, time.Now().UnixNano())))
	path := fmt.Sprintf("%s%s", config.SkinDir, filename)
	if err := c.SaveUploadedFile(file, path); err != nil {
		logger.Warning("皮肤上传失败: %s", err.Error())
		app.ThrowHttpException("皮肤上传失败...")
	}

	p := new(app.Player)
	p.Skin = filename

	if _, err := app.GetDb().Where("id = ?", uid).Cols("skin").Update(p); err != nil {
		logger.Warning("更新皮肤失败: %s", err.Error())

		//删除刚才上传的
		if err := os.Remove(path); err != nil {
			logger.Warning("新皮肤删除失败,可能不存在: %s", err.Error())
		}
		app.ThrowHttpException("更新皮肤失败,系统错误")
	} else {
		//删除旧皮肤
		if len(offinePlayer.Skin) > 0 {
			oldSkin := fmt.Sprintf("%s%s", config.SkinDir, offinePlayer.Skin)
			if err := os.Remove(oldSkin); err != nil {
				logger.Warning("旧皮肤删除失败,可能不存在: %s", err.Error())
			}
		}

	}
	c.JSON(200, app.BuildHttpResponse("更新皮肤成功", gin.H{
		"skin": fmt.Sprintf("%s%s", config.SkinUrl, filename),
	}))
}

//修改密码
func OffineEditPassword(c *gin.Context) {
	oldPwd := c.PostForm("old")
	newPwd := c.PostForm("new")
	if len(oldPwd) == 0 || len(newPwd) == 0 {
		app.ThrowHttpException("旧密码或新密码不能为空")
	}
	uid := c.GetString("uid")
	offinePlayer := app.Player{}
	app.GetDb().Where("id = ?", uid).Get(&offinePlayer)
	if utils.Md5String(oldPwd) != offinePlayer.Password {
		app.ThrowHttpException("旧密码不正确")
	}

	player := app.Player{}
	player.Password = utils.Md5String(newPwd)
	player.Sign = utils.Md5String(fmt.Sprintf("%v%v", time.Now().UnixNano(), uid))
	_, err := app.GetDb().Where("id = ?", uid).Cols("password", "sign").Update(player)
	if err != nil {
		logger.Warning("更新密码失败: %s", err.Error())
		app.ThrowHttpException("更新密码失败,系统错误")
	}
	c.JSON(200, app.BuildHttpResponse("更新密码成功,下次登录请使用新密码"))
}
