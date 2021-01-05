package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisCache struct {
	Rdb *redis.Client
	Ctx context.Context
}

var redisCache *RedisCache

func GetRedis() *RedisCache {
	return redisCache
}

func InitRedis(config *Config) error {
	r := new(RedisCache)

	r.Ctx = context.Background()
	conf := config.Redis
	r.Rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		DB:       conf.Index,
		Password: conf.Password,
	})
	err := r.Rdb.Ping(r.Ctx).Err()
	if err != nil {
		return errors.New(fmt.Sprintf("连接Redis失败: %s", err.Error()))
	}
	redisCache = r
	return nil
}

func (this *RedisCache) Set(name string, val interface{}, duration ...time.Duration) error {
	var t time.Duration = 0
	if len(duration) > 0 {
		t = duration[0]
	}
	return this.Rdb.Set(this.Ctx, name, val, t).Err()
}
func (this *RedisCache) Get(name string) (string, error) {
	return this.Rdb.Get(this.Ctx, name).Result()
}
func (this *RedisCache) Del(name string) (int64, error) {
	return this.Rdb.Del(this.Ctx, name).Result()
}
func (this *RedisCache) Exists(name string) bool {
	flag, err := this.Rdb.Exists(this.Ctx, name).Result()
	if err != nil {
		return false
	}
	if flag == 0 {
		return false
	}
	return true
}
func (this *RedisCache) GetMap(key, name string) (string, error) {
	return this.Rdb.HGet(this.Ctx, key, name).Result()
}
func (this *RedisCache) SetMap(key, name string, val interface{}) error {
	return this.Rdb.HSet(this.Ctx, key, name, val).Err()
}
func (this *RedisCache) ExistsMap(key, name string) bool {
	flag, err := this.Rdb.HExists(this.Ctx, key, name).Result()
	if err != nil {
		return false
	}
	return flag
}
