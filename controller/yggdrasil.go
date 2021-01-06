package controller

import (
	"encoding/base64"
	"fmt"
	"github.com/balrogsxt/minecraft-login/api"
	"github.com/balrogsxt/minecraft-login/app"
	"github.com/balrogsxt/minecraft-login/utils"
	"github.com/balrogsxt/minecraft-login/utils/logger"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

//登录redis储存数据
type PlayerTokenInfo struct {
	Uuid  string
	Token string
	Name  string
}
type PlayerServerInfo struct {
	Name     string
	Uuid     string
	ServerId string
}

//yggdrasil正版验证控制器

//1.首次认证服务器检测
func Index(c *gin.Context) {
	config := app.GetConfig()
	c.String(200, config.Server)
}

//2.客户端请求登录
func Login(c *gin.Context) {

	json := api.YggdrasilLogin{}
	if err := c.ShouldBind(&json); err != nil {
		app.ThrowYggdrasilException("登录失败", "参数错误")
	}

	var loginResult api.YggdrasilLoginResult

	//1.验证本地是否存在该用户
	var saveToken jwt.MapClaims
	offinePlayer := app.Player{}
	has, err := app.GetDb().Where("email = ?", json.Username).Get(&offinePlayer)
	if has {
		//1.离线登录
		md5Pwd := utils.Md5String(json.Password)
		if offinePlayer.Password != md5Pwd || len(json.Password) == 0 {
			app.ThrowYggdrasilException("验证失败", "离线玩家邮箱或密码错误!")
		}

		if len(offinePlayer.Uuid) == 0 || len(offinePlayer.Name) == 0 {
			app.ThrowYggdrasilException("验证角色失败", "没有可用的离线角色!")
		}

		selectedPlayer := api.MinecraftPlayer{
			UUID: offinePlayer.Uuid,
			Name: offinePlayer.Name,
		}

		loginResult = api.YggdrasilLoginResult{
			User: struct {
				Username string `json:"username"`
				Id       string `json:"id"`
			}{
				Username: offinePlayer.Email,
				Id:       fmt.Sprintf("%v", offinePlayer.Id),
			},
			ClientToken: json.ClientToken,
			AvailableProfiles: []api.MinecraftPlayer{
				selectedPlayer,
			},
			SelectedProfile: selectedPlayer,
		}
		saveToken = jwt.MapClaims{
			"uuid": offinePlayer.Uuid,
			"name": loginResult.SelectedProfile.Name,
			"type": "offine",
			"sign": offinePlayer.Sign, //只有离线才有
		}
	} else {
		//2.验证mojang账户,主要是获取正版账户的uuid等信息
		mojang := new(api.MojangAPI)
		loginResult = mojang.Login(json)
		saveToken = jwt.MapClaims{
			"uuid":        loginResult.SelectedProfile.UUID,
			"name":        loginResult.SelectedProfile.Name,
			"type":        "online",
			"mojangToken": loginResult.AccessToken, //正版登录包含一个mojangToken,用于刷新token使用
		}
	}
	saveToken["clientToken"] = json.ClientToken
	saveToken["timeout"] = fmt.Sprintf("%v", time.Now().Unix()+86400*30) //30天有效jwt token

	accessToken, err := utils.JwtBuild(saveToken)
	if err != nil {
		logger.Warning("创建jwt令牌失败: %s", err.Error())
		app.ThrowYggdrasilException("登录失败", "构建令牌失败")
	}
	loginResult.AccessToken = accessToken
	//直接返回mojang登录结果
	c.JSON(200, loginResult)
}

//3.客户端请求加入服务器
func Join(c *gin.Context) {
	json := api.YggdrasilJoin{}
	if err := c.ShouldBind(&json); err != nil {
		app.ThrowYggdrasilException("加入失败", "参数错误")
	}

	//验证玩家uuid是否对应token即可允许加入
	tokenMap, err := utils.JwtParse(json.AccessToken)
	if err != nil {
		app.ThrowYggdrasilException("加入失败", "验证数据错误,请尝试重新登录账号")
	}

	uuid := fmt.Sprintf("%v", tokenMap["uuid"])
	name := fmt.Sprintf("%v", tokenMap["name"])
	_type := fmt.Sprintf("%v", tokenMap["type"])
	if json.SelectedProfile != uuid {
		app.ThrowYggdrasilException("加入失败", "&c阁下账号验证失败,请重新在启动器端登录账号!")
	}

	if _type == "offine" {
		//离线玩家验证,验证该账户是否是sign相同即可
		offinePlayer := app.Player{}
		has, _ := app.GetDb().Where("`uuid` = ?", json.SelectedProfile).Get(&offinePlayer)
		if !has {
			app.ThrowYggdrasilException("ForbiddenOperationException", "账户数据错误,请尝试重新登录账号")
		}
		_sign := fmt.Sprintf("%v", tokenMap["sign"])
		if offinePlayer.Sign != _sign {
			app.ThrowYggdrasilException("ForbiddenOperationException", "账户可能已被更改,请尝试重新登录账号")
		}
	} else {
		mojangToken := fmt.Sprintf("%v", tokenMap["mojangToken"])
		json.AccessToken = mojangToken
		//正版验证,只尝试验证接口是否能够返回204或200即可验证成功
		mojang := new(api.MojangAPI)
		if err := mojang.Join(json); err != nil {
			app.ThrowYggdrasilException("加入失败", "&c正版账户身份验证失败,请尝试重新登录!")
		}
	}

	//设置redis这个昵称的玩家加入了指定serverId服务器,后续用于验证
	serverInfo := PlayerServerInfo{
		Uuid:     json.SelectedProfile,
		Name:     name,
		ServerId: json.ServerId,
	}
	//10秒内有效信息
	field := fmt.Sprintf("joinServer:%s", utils.Md5String(name+json.ServerId))
	if err := app.GetRedis().Set(field, utils.JsonEncode(serverInfo)); err != nil {
		app.ThrowYggdrasilException("加入失败", "&c加入失败,请重试!")
	}
	c.JSON(204, gin.H{})
}

//4.服务端验证是否允许加入
func HasJoined(c *gin.Context) {
	username := c.Query("username")
	serverId := c.Query("serverId")

	//获取对应的redis信息
	field := fmt.Sprintf("joinServer:%s", utils.Md5String(username+serverId))
	json, _ := app.GetRedis().Get(field)
	serverInfo := PlayerServerInfo{}
	if err := utils.JsonDecode(json, &serverInfo); err != nil {
		logger.Warning("验证服务端加入请求失败: %s", err.Error())
		app.ThrowYggdrasilException("验证失败", "服务器数据错误")
	}
	if len(username) == 0 || len(serverId) == 0 || len(serverInfo.ServerId) == 0 || len(serverInfo.Uuid) == 0 {
		app.ThrowYggdrasilException("验证失败", "验证失败,玩家角色参数错误")
	}
	if username != serverInfo.Name || serverId != serverInfo.ServerId {
		app.ThrowYggdrasilException("验证失败", "验证失败,玩家角色无法匹配")
	}

	c.JSON(200, api.MinecraftPlayer{
		UUID: serverInfo.Uuid,
		Name: serverInfo.Name,
	})
}

//其他API
//验证token是否可用
func Validate(c *gin.Context) {
	json := api.YggdrasilValidate{}
	if err := c.ShouldBind(&json); err != nil {
		app.ThrowYggdrasilException("ForbiddenOperationException", "参数错误")
	}

	//验证玩家uuid是否对应token即可允许加入
	tokenMap, err := utils.JwtParse(json.AccessToken)
	if err != nil {
		app.ThrowYggdrasilException("ForbiddenOperationException", "验证数据错误,请尝试重新登录账号")
	}
	mode := fmt.Sprintf("%v", tokenMap["type"])
	uuid := fmt.Sprintf("%v", tokenMap["uuid"])
	if mode == "online" {
		at := fmt.Sprintf("%v", tokenMap["mojangToken"])
		//正版
		mojang := new(api.MojangAPI)
		if ex := mojang.Validate(at, json.ClientToken); ex != nil {
			panic(ex)
		}
	} else {
		//离线
		timeout, er := strconv.ParseInt(fmt.Sprintf("%s", tokenMap["timeout"]), 10, 64)
		if er != nil {
			app.ThrowYggdrasilException("ForbiddenOperationException", "数据错误,请尝试重新登录账号:"+er.Error())
		}
		if time.Now().Unix() > timeout {
			app.ThrowYggdrasilException("ForbiddenOperationException", "账号登录已过期,请尝试重新登录账号")
		}

		//如果账户的sign与当前账户实际的sign不相同,需要重新输入密码
		offinePlayer := app.Player{}
		has, _ := app.GetDb().Where("`uuid` = ?", uuid).Get(&offinePlayer)
		if !has {
			app.ThrowYggdrasilException("ForbiddenOperationException", "账户数据错误,请尝试重新登录账号")
		}
		_sign := fmt.Sprintf("%v", tokenMap["sign"])
		if offinePlayer.Sign != _sign {
			app.ThrowYggdrasilException("ForbiddenOperationException", "账户可能已被更改,请尝试重新登录账号")
		}

		c.Status(204)
	}
}
func Refresh(c *gin.Context) {

	json := api.YggdrasilRefresh{}
	if err := c.ShouldBind(&json); err != nil {
		app.ThrowYggdrasilException("ForbiddenOperationException", "参数错误")
	}
	//验证玩家uuid是否对应token即可允许加入
	tokenMap, err := utils.JwtParse(json.AccessToken)
	if err != nil {
		app.ThrowYggdrasilException("ForbiddenOperationException", "验证数据错误,请尝试重新登录账号")
	}
	mode := fmt.Sprintf("%v", tokenMap["type"])
	uuid := fmt.Sprintf("%v", tokenMap["uuid"])

	var saveToken jwt.MapClaims
	var loginResult api.YggdrasilLoginResult
	if mode == "online" {
		//正版
		mojang := new(api.MojangAPI)

		at := fmt.Sprintf("%v", tokenMap["mojangToken"])
		json.AccessToken = at
		loginResult = mojang.Refresh(json)
		saveToken = jwt.MapClaims{
			"uuid":        loginResult.SelectedProfile.UUID,
			"name":        loginResult.SelectedProfile.Name,
			"type":        "online",
			"mojangToken": loginResult.AccessToken, //正版登录包含一个mojangToken,用于刷新token使用
		}
	} else {
		//离线需要查询角色信息
		offinePlayer := app.Player{}
		has, _ := app.GetDb().Where("`uuid` = ?", uuid).Get(&offinePlayer)
		if !has {
			app.ThrowYggdrasilException("ForbiddenOperationException", "账户数据错误,请尝试重新登录账号")
		}
		_sign := fmt.Sprintf("%v", tokenMap["sign"])
		if offinePlayer.Sign != _sign {
			app.ThrowYggdrasilException("ForbiddenOperationException", "账户可能已被更改,请尝试重新登录账号")
		}

		timeout, er := strconv.ParseInt(fmt.Sprintf("%s", tokenMap["timeout"]), 10, 64)
		if er != nil {
			app.ThrowYggdrasilException("ForbiddenOperationException", "数据错误,请尝试重新登录账号:"+er.Error())
		}

		if time.Now().Unix() > timeout {
			app.ThrowYggdrasilException("ForbiddenOperationException", "1账号登录已过期,请尝试重新登录账号")
		}
		if time.Now().Unix()+86400*7 > timeout { //登录过期超过7天,进入永久失效状态
			app.ThrowYggdrasilException("ForbiddenOperationException", "登录信息已永久过期,请尝试重新登录账号")
		}

		//离线
		selectedPlayer := api.MinecraftPlayer{
			UUID: offinePlayer.Uuid,
			Name: offinePlayer.Name,
		}

		loginResult = api.YggdrasilLoginResult{
			User: struct {
				Username string `json:"username"`
				Id       string `json:"id"`
			}{
				Username: offinePlayer.Email,
				Id:       fmt.Sprintf("%v", offinePlayer.Id),
			},
			ClientToken: json.ClientToken,
			AvailableProfiles: []api.MinecraftPlayer{
				selectedPlayer,
			},
			SelectedProfile: selectedPlayer,
		}
		saveToken = jwt.MapClaims{
			"uuid": offinePlayer.Uuid,
			"name": loginResult.SelectedProfile.Name,
			"type": "offine",
			"sign": offinePlayer.Sign, //只有离线才有
		}
	}
	saveToken["clientToken"] = json.ClientToken
	saveToken["timeout"] = fmt.Sprintf("%v", time.Now().Unix()+86400*30) //30天有效jwt token
	accessToken, er := utils.JwtBuild(saveToken)
	if er != nil {
		app.ThrowYggdrasilException("ForbiddenOperationException", "获取登录数据失败,请尝试重新登录账号")
	}
	loginResult.AccessToken = accessToken
	//直接返回mojang登录结果
	c.JSON(200, loginResult)
}

//获取玩家皮肤
func GetSkin(c *gin.Context) {
	uuid := c.Param("uuid")

	//查询本地是否有这个uuid的玩家
	offinePlayer := app.Player{}

	if has, _ := app.GetDb().Where("`uuid` = ?", uuid).Get(&offinePlayer); has {
		//离线自定义皮肤

		if len(offinePlayer.Skin) == 0 {
			//没有皮肤
			c.Status(404)
			return
		}
		config := app.GetConfig()
		_ = config
		skinUrl := fmt.Sprintf("%s%s", config.SkinUrl, offinePlayer.Skin)

		skinJson := utils.JsonEncode(gin.H{
			"timestamp":   time.Now().Unix() - 86400,
			"profileId":   offinePlayer.Uuid,
			"profileName": offinePlayer.Name,
			"isPublic":    true,
			"textures": gin.H{
				"SKIN": gin.H{
					"url": skinUrl,
					"metadata": gin.H{
						"model": "slim",
					},
				},
			},
		})

		skinBase64 := base64.StdEncoding.EncodeToString([]byte(skinJson))

		c.JSON(200, api.PlayerProfileResult{
			UUID: offinePlayer.Uuid,
			Name: offinePlayer.Name,
			Properties: []api.PlayerProfile{
				{
					Name:  "textures",
					Value: skinBase64,
				},
			},
		})
	} else {
		//正版皮肤
		mojang := new(api.MojangAPI)
		c.JSON(200, mojang.GetSkin(uuid))
	}

}
