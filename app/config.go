package app

import (
	"errors"
	"fmt"
	"github.com/balrogsxt/minecraft-login/utils/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Mysql struct {
		Host     string
		Port     int
		Name     string
		User     string
		Password string
	}
	Redis struct {
		Host     string
		Port     int
		Password string
		Index    int
	}
	Server   string
	HttpPort string `yaml:"http_port"`
	SkinDir  string `yaml:"skin_dir"`
	SkinUrl  string `yaml:"skin_url"`
	JwtKey   string `yaml:"jwt_key"`
}

func LoadConfig() (*Config, error) {
	file := "./config.yml"
	_byte, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	conf := Config{}
	if err := yaml.Unmarshal(_byte, &conf); err != nil {
		return nil, errors.New(fmt.Sprintf("解析配置文件失败: %s", err.Error()))
	}
	b, er := ioutil.ReadFile("./server.json")
	if er != nil {
		return nil, errors.New(fmt.Sprintf("验证服务器信息未配置: %s", er.Error()))
	}
	conf.Server = string(b)

	config = &conf
	return &conf, nil
}

var config *Config

func GetConfig() *Config {
	if config == nil {
		conf, err := LoadConfig()
		if err != nil {
			logger.Fatal("[配置文件] 加载失败: %s", err.Error())
			return nil
		}
		return conf
	} else {
		return config
	}
}
