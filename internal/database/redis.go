// internal/database/redis.go - PRODUCTION READY

package database

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "strings"
    "time"

    "github.com/redis/go-redis/v9"
)

type RedisDB struct {
    Client *redis.Client
    Ctx    context.Context
}

func NewRedis(addr string) *RedisDB {
    var rdb *redis.Client
    
    // Check if addr is a full URL (redis://...)
    if strings.Contains(addr, "://") {
        // Parse URL
        opt, err := redis.ParseURL(addr)
        if err != nil {
            log.Printf("Redis URL parse error: %v", err)
            // Fallback to simple addr
            rdb = redis.NewClient(&redis.Options{
                Addr: addr,
            })
        } else {
            rdb = redis.NewClient(opt)
        }
    } else {
        // Simple host:port
        rdb = redis.NewClient(&redis.Options{
            Addr:     addr,
            Password: os.Getenv("REDIS_PASSWORD"),
            DB:       0,
        })
    }
    
    // Test connection
    ctx := context.Background()
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Printf("Redis connection warning: %v", err)
    } else {
        log.Println("Redis connected successfully")
    }
    
    return &RedisDB{
        Client: rdb,
        Ctx:    context.Background(),
    }
}

func (r *RedisDB) Close() {
    if r.Client != nil {
        r.Client.Close()
    }
}

// CacheData with proper duration
func (r *RedisDB) CacheData(key string, value interface{}, duration time.Duration) {
    if duration > 0 && duration < time.Second {
        duration = duration * time.Second
    }
    if duration == 0 {
        duration = 5 * time.Minute
    }
    
    data, err := json.Marshal(value)
    if err != nil {
        log.Printf("Redis marshal error: %v", err)
        return
    }
    
    err = r.Client.Set(r.Ctx, key, data, duration).Err()
    if err != nil {
        log.Printf("Redis set error: %v", err)
    }
}

func (r *RedisDB) GetCachedData(key string, dest interface{}) error {
    data, err := r.Client.Get(r.Ctx, key).Result()
    if err != nil {
        return err
    }
    return json.Unmarshal([]byte(data), dest)
}

func (r *RedisDB) DeleteCache(key string) {
    r.Client.Del(r.Ctx, key)
}

func (r *RedisDB) Ping() error {
    return r.Client.Ping(r.Ctx).Err()
}