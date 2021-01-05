package api

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/balrogsxt/minecraft-login/app"
	"github.com/balrogsxt/minecraft-login/utils/logger"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"strings"
)

const (
	MojangAuthServer string = "https://authserver.mojang.com"    //mojang验证服务器
	LoginPath               = MojangAuthServer + "/authenticate" //登录地址
	ValidatePath            = MojangAuthServer + "/validate"     //验证地址
	RefreshPath             = MojangAuthServer + "/refresh"      //重获地址
	GetSkinUrl              = "https://sessionserver.mojang.com/session/minecraft/profile/"
	JoinServer              = "https://sessionserver.mojang.com/session/minecraft/join"
)

type MinecraftPlayer struct {
	UUID string `json:"id"`
	Name string `json:"name"`
}
type YggdrasilValidate struct {
	AccessToken string `json:"accessToken"`
	ClientToken string `json:"clientToken"`
}

//登录客户端请求数据
type YggdrasilLogin struct {
	Agent struct {
		Name    string
		Version int
	}
	RequestUser bool `json:"requestUser"`

	Username    string //登录邮箱
	Password    string //登录密码
	ClientToken string `json:"clientToken"` //客户端token
}
type YggdrasilRefresh struct {
	AccessToken string `yaml:"accessToken"`
	ClientToken string `yaml:"clientToken"`
	RequestUser bool   `yaml:"requestUser"`
}

//登录服务端返回数据
type YggdrasilLoginResult struct {
	User struct {
		Username string `json:"username"`
		Id       string `json:"id"`
	} `json:"user"`
	AccessToken       string            `json:"accessToken"`
	ClientToken       string            `json:"clientToken"`
	AvailableProfiles []MinecraftPlayer `json:"availableProfiles"` //可选的玩家列表
	SelectedProfile   MinecraftPlayer   `json:"selectedProfile"`   //当前选择的角色
}

//客户端加入服务器
type YggdrasilJoin struct {
	AccessToken     string `json:"accessToken"`     //登录返回的Token
	SelectedProfile string `json:"selectedProfile"` //选择的玩家UUID
	ServerId        string `json:"serverId"`        //客户端选择的id
}
type YggdrasilHasJoined struct {
	Username string `json:"username"` //玩家用户名,不是UUID
	ServerId string `json:"serverId"` //客户端选择的id
}

//角色材质,皮肤
type PlayerProfile struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//查询皮肤返回结果
type PlayerProfileResult struct {
	UUID       string          `json:"id"`
	Name       string          `json:"name"`
	Properties []PlayerProfile `json:"properties"`
}

type MojangAPI struct {
}

func (this *MojangAPI) Login(json YggdrasilLogin) YggdrasilLoginResult {
	_json := req.BodyJSON(gin.H{
		"agent": gin.H{
			"name":    json.Agent.Name,
			"version": json.Agent.Version,
		},
		"username":    json.Username,
		"password":    json.Password,
		"requestUser": json.RequestUser,
		"clientToken": json.ClientToken,
	})

	header := req.Header{
		"Content-Type": "application/json",
	}
	res, err := req.Post(LoginPath, _json, header)
	if err != nil {
		logger.Warning("请求Mojang服务器失败: %s", err.Error())
		app.ThrowYggdrasilException("登录失败", "请求Mojang服务器失败")
	}

	m := gin.H{}
	if err := res.ToJSON(&m); err != nil {
		app.ThrowYggdrasilException("登录失败:", "请求Mojang服务器解析数据失败")
	}
	_, isError := m["error"]
	if isError {
		if strings.Contains(fmt.Sprintf("%v", m["errorMessage"]), "Invalid username or password") {
			m["error"] = "正版账户验证失败"
			m["errorMessage"] = "邮箱或密码错误"
		}
		logger.Info("%#v ", m)
		app.ThrowYggdrasilException(m["error"], m["errorMessage"])
	}
	result := YggdrasilLoginResult{}
	if err := res.ToJSON(&result); err != nil {
		app.ThrowYggdrasilException("登录失败:", "解析数据失败")
	}
	return result
}

//请求加入服务器(在本项目中仅作为验证...不做hasJoined判断
func (this *MojangAPI) Join(json YggdrasilJoin) error {
	_json := req.BodyJSON(gin.H{
		"accessToken":     json.AccessToken,
		"selectedProfile": json.SelectedProfile,
		"serverId":        json.ServerId,
	})
	header := req.Header{
		"Content-Type": "application/json",
	}
	res, err := req.Post(JoinServer, _json, header)
	if err != nil {
		logger.Warning("请求Mojang服务器失败: %s", err.Error())
		app.ThrowYggdrasilException("登录失败", "请求Mojang服务器失败")
	}
	code := res.Response().StatusCode
	if code == 204 || code == 200 {
		return nil
	} else {
		return errors.New("ForbiddenOperationException")
	}

}
func (this *MojangAPI) GetSkin(uuid string) PlayerProfileResult {
	res, err := req.Get(fmt.Sprintf("%s%s?unsigned=false", GetSkinUrl, uuid))
	if err != nil {
		app.ThrowYggdrasilException("获取失败", "获取皮肤失败")
	}
	result := PlayerProfileResult{}
	if err := res.ToJSON(&result); err != nil {
		app.ThrowYggdrasilException("获取失败", "获取皮肤失败,解析失败")
	}
	return result
}

func (this *MojangAPI) Validate(accessToken, clientToken string) *app.YggdrasilException {
	_json := req.BodyJSON(gin.H{
		"accessToken": accessToken,
		"clientToken": clientToken,
	})

	header := req.Header{
		"Content-Type": "application/json",
	}
	res, err := req.Post(ValidatePath, _json, header)
	if err != nil {
		logger.Warning("请求Mojang服务器失败: %s", err.Error())
		app.ThrowYggdrasilException("ForbiddenOperationException", "请求Mojang服务器失败")
	}
	statusCode := res.Response().StatusCode
	if statusCode == 200 || statusCode == 204 {
		return nil
	}
	ex := app.YggdrasilException{}
	res.ToJSON(&ex)
	return &ex
}

func (this *MojangAPI) Refresh(json YggdrasilRefresh) YggdrasilLoginResult {
	_json := req.BodyJSON(gin.H{
		"accessToken": json.AccessToken,
		"clientToken": json.ClientToken,
		"requestUser": json.RequestUser,
	})
	header := req.Header{
		"Content-Type": "application/json",
	}
	res, err := req.Post(RefreshPath, _json, header)
	if err != nil {
		logger.Warning("请求Mojang服务器失败: %s", err.Error())
		app.ThrowYggdrasilException("ForbiddenOperationException", "请求Mojang服务器失败")
	}

	result := YggdrasilLoginResult{}
	if err := res.ToJSON(&result); err != nil {
		app.ThrowYggdrasilException("登录失败:", "解析数据失败")
	}
	return result

}

//计算离线玩家UUID
func GetOffinePlayerUUID(name string) string {
	playerName := fmt.Sprintf("OfflinePlayer:%s", name)
	m := md5.New()
	m.Write([]byte(playerName))
	byte := m.Sum(nil)
	byte[6] &= 0x0f
	byte[6] |= 0x30
	byte[8] &= 0x3f
	byte[8] |= 0x80
	return hex.EncodeToString(byte)
}
