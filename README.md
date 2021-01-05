# 简单自定义皮肤验证登录

## 简介
主要用于简单快速搭建一个支持正版玩家、离线玩家共同登录的外置登录方案(该方案主要用于人少的朋友服)

## 项目由来
主要用于朋友之前的服务器、可能有些玩家有正版，而有些玩家没有正版，在高版本中离线服务器无法加载正版的皮肤方案(插件、MOD除外)
不想使用其他第三方皮肤站方案，就可以使用本项目实现正版玩家正版账户密码登录，离线玩家独立注册设置皮肤

## 储存
项目内mysql主要用于储存离线玩家的数据、redis储存加入服务器验证消息,正版玩家accessToken存在jwt加密中，并不会在服务端储存
登录有效期30天,到期后进入7天暂时失效状态,之后不续签则进入永久失效，正版玩家根据mojang官方处理
离线玩家修改昵称、密码都会变更sign值，故客户端登录过的账户需要重新登录

## 玩家登录方式
> 正版玩家 -> 使用正版账户登录即可
> 离线玩家 -> 注册账号后绑定昵称、皮肤


## 项目配置
> 运行根目录下创建`config.yml`作为配置文件
```
http_port: 10002 #启动端口
jwt_key: "12345667" #jwt密钥
skin_dir: "/home/xxxx/xxx/" #皮肤上传保存的目录
skin_url: "http://xxx.com/" #访问皮肤的URL地址目录
mysql: #mysql配置
  host: ""
  port: 3306
  name: "minecraft"
  user: root
  password: "password"   
redis: #redis配置
  host: 127.0.0.1
  port: 6379
  password: ""
  index: 3

```
> 数据表导入
```
CREATE TABLE `player` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `uuid` varchar(50) NOT NULL COMMENT '离线玩家UUID',
  `email` varchar(50) DEFAULT NULL COMMENT '离线玩家邮箱地址',
  `name` varchar(50) DEFAULT NULL COMMENT '离线玩家玩家名称',
  `skin` varchar(255) DEFAULT NULL COMMENT '皮肤保存地址',
  `time` int(10) DEFAULT NULL COMMENT '创建时间',
  `password` varchar(50) DEFAULT NULL COMMENT 'md5密码,不做盐了,麻烦',
  `sign` varchar(50) DEFAULT NULL COMMENT '发生密码修改、修改昵称会更变,则账户token需要重新登录',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `uuid` (`uuid`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='离线玩家数据表';
```
> 根目录的验证服务器信息配置`server.json`
```
{
    "meta": {
        "serverName": "xxxx",
        "implementationName": "yggdrasil api",
        "implementationVersion": "1.0.0",
        "links": {
            "homepage":"",
            "register": ""
        }
    },
    "feature.non_email_login": true
    //其他配置项
}
```

## 客户端使用方法
> 推荐使用HMCL启动器
- 1.选择添加角色
- 2.选择添加认证服务器
- 3.输入需要添加的认证服务器地址【即本项目启动后的地址】
- 4.正版玩家使用正版账号密码登录、离线玩家注册后登录

## 服务端使用方法
- 服务端通过配置 [authlib injector](https://github.com/yushijinhun/authlib-injector/wiki/%E5%9C%A8-Minecraft-%E6%9C%8D%E5%8A%A1%E7%AB%AF%E4%BD%BF%E7%94%A8-authlib-injector) 进行处理
## 可能存在的问题
正版玩家角色名称与离线玩家角色名称冲突问题，由于用于朋友服，故忽略该问题