package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient Redis客户端
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(addr string, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close 关闭Redis连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Set 设置键值对
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	if val, ok := value.(string); ok {
		return r.client.Set(r.ctx, key, val, expiration).Err()
	}

	// 尝试JSON序列化
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}

	return r.client.Set(r.ctx, key, jsonData, expiration).Err()
}

// Get 获取值
func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

// GetJSON 获取JSON数据并反序列化
func (r *RedisClient) GetJSON(key string, dest interface{}) error {
	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// Delete 删除键
func (r *RedisClient) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(key string) (bool, error) {
	exists, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Expire 设置过期时间
func (r *RedisClient) Expire(key string, expiration time.Duration) error {
	return r.client.Expire(r.ctx, key, expiration).Err()
}

// Incr 递增
func (r *RedisClient) Incr(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

// Decr 递减
func (r *RedisClient) Decr(key string) (int64, error) {
	return r.client.Decr(r.ctx, key).Result()
}

// LPush 左侧推入列表
func (r *RedisClient) LPush(key string, values ...interface{}) error {
	return r.client.LPush(r.ctx, key, values...).Err()
}

// RPop 右侧弹出列表
func (r *RedisClient) RPop(key string) (string, error) {
	return r.client.RPop(r.ctx, key).Result()
}

// LLen 获取列表长度
func (r *RedisClient) LLen(key string) (int64, error) {
	return r.client.LLen(r.ctx, key).Result()
}

// SAdd 添加集合元素
func (r *RedisClient) SAdd(key string, members ...interface{}) error {
	return r.client.SAdd(r.ctx, key, members...).Err()
}

// SRem 移除集合元素
func (r *RedisClient) SRem(key string, members ...interface{}) error {
	return r.client.SRem(r.ctx, key, members...).Err()
}

// SIsMember 检查是否是集合成员
func (r *RedisClient) SIsMember(key string, member interface{}) (bool, error) {
	return r.client.SIsMember(r.ctx, key, member).Result()
}

// HSet 设置哈希字段
func (r *RedisClient) HSet(key string, field string, value interface{}) error {
	return r.client.HSet(r.ctx, key, field, value).Err()
}

// HGet 获取哈希字段
func (r *RedisClient) HGet(key string, field string) (string, error) {
	return r.client.HGet(r.ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	return r.client.HGetAll(r.ctx, key).Result()
}

// HDel 删除哈希字段
func (r *RedisClient) HDel(key string, fields ...string) error {
	return r.client.HDel(r.ctx, key, fields...).Err()
}

// ZAdd 添加有序集合元素
func (r *RedisClient) ZAdd(key string, score float64, member string) error {
	return r.client.ZAdd(r.ctx, key, &redis.Z{Score: score, Member: member}).Err()
}

// ZRem 移除有序集合元素
func (r *RedisClient) ZRem(key string, members ...string) error {
	return r.client.ZRem(r.ctx, key, members...).Err()
}

// ZRange 获取有序集合范围内的元素
func (r *RedisClient) ZRange(key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(r.ctx, key, start, stop).Result()
}

// ZRevRange 获取有序集合范围内的元素（降序）
func (r *RedisClient) ZRevRange(key string, start, stop int64) ([]string, error) {
	return r.client.ZRevRange(r.ctx, key, start, stop).Result()
}

// ZScore 获取有序集合元素的分数
func (r *RedisClient) ZScore(key string, member string) (float64, error) {
	return r.client.ZScore(r.ctx, key, member).Result()
}

// ZRank 获取有序集合元素的排名（升序）
func (r *RedisClient) ZRank(key string, member string) (int64, error) {
	return r.client.ZRank(r.ctx, key, member).Result()
}

// ZRevRank 获取有序集合元素的排名（降序）
func (r *RedisClient) ZRevRank(key string, member string) (int64, error) {
	return r.client.ZRevRank(r.ctx, key, member).Result()
}
