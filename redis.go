package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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

func (r RedisConn) doInt(commandName string, args ...interface{}) (reply int, err error) {
	n, err := redis.Int(r.do(commandName, args...))
	return n, err
}

func (r RedisConn) doInt64(commandName string, args ...interface{}) (reply int64, err error) {
	n, err := redis.Int64(r.do(commandName, args...))
	return n, err
}

func (r RedisConn) doString(commandName string, args ...interface{}) (reply string, err error) {
	n, err := redis.String(r.do(commandName, args...))
	return n, err
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
	if PersonID == 0 {
		return errors.New("Invalid person's or list's ID")
	}
	if ListID >= endofwl || ListID < 0 {
		return errors.New("ListID is invalid")
	}

	r.key = fmt.Sprintf("plist:%d", PersonID)
	r.value = ListID
	return r.Set(context.Background()) // TODO: use context in the future
}

// Set last action of a person to the cache
func (r *RedisSetRequest) SetLastAction(PersonID int64, action LastAction) error {
	if PersonID == 0 {
		return errors.New("Invalid person's ID")
	}

	r.key = fmt.Sprintf("lastact:%d", PersonID)
	r.value = action
	r.expireInSec = 3600               // one hour for making an action
	return r.Set(context.Background()) // TODO: use context in the future
}

// Set number of words in the generated passwords for the person
func (r *RedisSetRequest) SetNumberOfWords(PersonID int64, n int) error {
	if PersonID == 0 {
		return errors.New("Invalid person's ID")
	}

	r.key = fmt.Sprintf("wordsn:%d", PersonID)
	r.value = n
	r.expireAt = time.Now().Add(365 * 24 * time.Hour) // To free some memory after a year
	return r.Set(context.Background())                // TODO: use context in the future
}

// Set separator for the generated passwords for the person
func (r *RedisSetRequest) SetSeparator(PersonID int64, s string) error {
	if PersonID == 0 {
		return errors.New("Invalid person's ID")
	}

	r.key = fmt.Sprintf("sep:%d", PersonID)
	r.value = s
	r.expireAt = time.Now().Add(365 * 24 * time.Hour) // To free some memory after a year
	return r.Set(context.Background())                // TODO: use context in the future
}

type RedisGetRequest struct {
	conn RedisConn
	id   int64  // any id as a part of redis key (after colon)
	key  string // use key instead of id
}

func (r RedisConn) NewRedisGetRequest() *RedisGetRequest {
	return func() *RedisGetRequest {
		return &RedisGetRequest{
			conn: r,
		}
	}()
}

// Set key for request
func (r *RedisGetRequest) Key(k string) *RedisGetRequest {
	r.key = k
	return r
}

// Set ID for request
func (r *RedisGetRequest) ID(id int64) *RedisGetRequest {
	r.id = id
	return r
}

// Get listID of person. Returns 0 if not found
func (r *RedisGetRequest) GetPersonList() WL {
	if r.id == 0 {
		n, err := r.conn.doInt("GET", r.key)
		if err != nil {
			log.Println("ERROR:", err)
			return 0
		}
		return WL(n)
	}

	n, err := r.conn.doInt("GET", fmt.Sprintf("plist:%d", r.id))
	if err != nil {
		return 0
	}
	return WL(n)
}

func (r *RedisGetRequest) GetLastAction() (LastAction, error) {
	la, err := r.conn.doString("GET", fmt.Sprintf("lastact:%d", r.id))
	return LastAction(la), err
}

func (r *RedisGetRequest) GetWordsNumber() (int, error) {
	n, err := r.conn.doInt("GET", fmt.Sprintf("wordsn:%d", r.id))
	return n, err
}

func (r *RedisGetRequest) GetSeparator() (string, error) {
	s, err := r.conn.doString("GET", fmt.Sprintf("sep:%d", r.id))
	return s, err
}

type RedisDelRequest struct {
	conn RedisConn
	id   int64  // any id as a part of redis key (after colon)
	key  string // use key instead of id
}

func (r RedisConn) NewRedisDelRequest() *RedisDelRequest {
	return func() *RedisDelRequest {
		return &RedisDelRequest{
			conn: r,
		}
	}()
}

// Set key for request
func (r *RedisDelRequest) Key(k string) *RedisDelRequest {
	r.key = k
	return r
}

// Set ID for request
func (r *RedisDelRequest) ID(id int64) *RedisDelRequest {
	r.id = id
	return r
}

// Exec is used to delete value from cache.
//
// You have to specify Redis connection and key to use this function
func (r *RedisDelRequest) Exec() error {
	_, err := r.conn.do("DEL", r.key)
	return err
}

// You have to specify conn and id in order to use this function
func (r *RedisDelRequest) DeleteLastAction() error {
	if r.id == 0 {
		return errors.New("You have to specify id of a person")
	}
	r.Key(fmt.Sprintf("lastact:%d", r.id))
	err := r.Exec()
	return err
}
