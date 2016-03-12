package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
)

type Backend interface {
	Close()
	AddRoom(room string) error
	AddMessage(room string, name string, msg string) error

	GetRoom(room string) (Room, []Message, error)
	GetRooms() ([]Room, error)
}

type Message struct {
	Room    string
	MsgId   string
	Text    string
	Author  string
	Created time.Time
}

type Room struct {
	RoomId string
	Href   string
}

type RedisBackend struct {
	Redis redis.Conn
}

func NewRedisBackend(hostport string) (Backend, error) {
	conn, err := redis.Dial("tcp", hostport)
	if err != nil {
		return nil, err
	}
	return &RedisBackend{conn}, nil
}

func (backend *RedisBackend) Close() {
	backend.Redis.Close()
}

func (backend *RedisBackend) AddRoom(room string) error {
	backend.Redis.Do("SADD", "rooms", room)
	return nil
}

func (backend *RedisBackend) AddMessage(room string, name string, msg string) error {
	nextId, err := redis.Int64(backend.Redis.Do("INCR", "message:id"))
	if err != nil {
		return err
	}
	msgId := fmt.Sprintf("message:%d", nextId)
	backend.Redis.Do("HMSET", msgId, "author", name, "room", room, "text", msg, "time", time.Now())
	backend.Redis.Do("RPUSH", room+":messages", msgId)
	return nil
}

func (backend *RedisBackend) GetRoom(room string) (Room, []Message, error) {
	messageIds, _ := redis.Strings(backend.Redis.Do("LRANGE", room+":messages", 0, -1))
	messages := []Message{}

	for _, msgId := range messageIds {
		res, _ := redis.Strings(backend.Redis.Do("HMGET", msgId, "text", "author", "time"))
		t, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", res[2])
		msg := Message{Room: room, MsgId: msgId, Text: res[0], Author: res[1], Created: t}
		messages = append(messages, msg)
	}

	return Room{room, ""}, messages, nil
}

func (backend *RedisBackend) GetRooms() ([]Room, error) {
	rooms := []Room{}
	roomIds, _ := redis.Strings(backend.Redis.Do("SMEMBERS", "rooms"))
	for _, roomId := range roomIds {
		room := Room{roomId, "/api/rooms/" + roomId}
		rooms = append(rooms, room)
	}
	return rooms, nil
}
