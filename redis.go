package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisConn struct {
	conn redis.Conn
}

func NewConn(pool *redis.Pool) RedisConn {
	return func() RedisConn {
		return RedisConn{
			conn: pool.Get(),
		}
	}()
}

func (c RedisConn) Close() {
	c.conn.Close()
}

type RedisRequest struct {
	conn       RedisConn
	command    string
	params     []interface{}
	wantResult bool
}

func (r RedisConn) NewRedisRequest() *RedisRequest {
	return func() *RedisRequest {
		return &RedisRequest{
			conn: r,
		}
	}()
}

func (r RedisConn) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return r.conn.Do(commandName, args...)
}

// Change command of the redis request struct
func (r *RedisRequest) Command(c string) {

}

// Add params into redis request struct
func (r *RedisRequest) Params(p []interface{}) {

}

// Set wantResult value into redis request struct
func (r *RedisRequest) WantResult(w bool) {

}

func (r *RedisRequest) GetBool(ctx context.Context) (bool, error) {
	panic("Not implemented")
}
func (r *RedisRequest) GetInt(ctx context.Context) (bool, error) {
	panic("Not implemented")
}
func (r *RedisRequest) GetString(ctx context.Context) (bool, error) {
	panic("Not implemented")
}
func (r *RedisRequest) GetStringSlice(ctx context.Context) (bool, error) {
	panic("Not implemented")
}

type RedisSetRequest struct {
	conn        RedisConn
	key         string
	value       interface{}
	expireInSec int       // Use one of the expiration parameters
	expireAt    time.Time // Will be used only if expireInSec is 0
}

func (r RedisConn) NewRedisSetRequest() *RedisSetRequest {
	return func() *RedisSetRequest {
		return &RedisSetRequest{
			conn: r,
		}
	}()
}

// Set key of the redis set request struct
func (r *RedisSetRequest) Key(k string) *RedisSetRequest {
	r.key = k
	return r
}

// Add value into redis set request struct
func (r *RedisSetRequest) Value(v interface{}) *RedisSetRequest {
	r.value = v
	return r
}

// Set anount of seconds until value will be destroyed
func (r *RedisSetRequest) ExpireIn(s int) *RedisSetRequest {
	r.expireInSec = s
	return r
}

// Set expiration time (when the value will expire)
func (r *RedisSetRequest) ExpireAt(t time.Time) *RedisSetRequest {
	r.expireAt = t
	return r
}

func (r *RedisSetRequest) Set(ctx context.Context) error {
	// Without expiration
	if r.expireInSec == 0 && r.expireAt.IsZero() {
		_, err := r.conn.do("SET", r.key, r.value)
		return err
	}

	// With expiration
	// Get number of seconds until expiration
	s := func() int {
		if r.expireInSec > 0 {
			return r.expireInSec
		}
		if r.expireAt.After(time.Now()) {
			return int(r.expireAt.Sub(time.Now()).Seconds())
		}
		return -2
	}()

	if s < 0 {
		return errors.New("Expiration time is less than zero")
	}

	_, err := r.conn.do("SETEX", r.key, s, r.value)
	return err
}

// Set desired wordlist of a person to the cache
func (r *RedisSetRequest) SetPersonList(PersonID int64, ListID WL) error {
	if PersonID == 0 || ListID == 0 {
		return errors.New("Invalid person's or list's ID")
	}
	if ListID >= endofwl || ListID < 0 {
		return errors.New("ListID is invalid")
	}

	r.key = fmt.Sprintf("plist:%d", PersonID)
	r.value = ListID
	return r.Set(context.Background()) // TODO: use context in the future
}
