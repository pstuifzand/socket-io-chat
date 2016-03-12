package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	socketio "github.com/googollee/go-socket.io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type AdminMessage struct {
	Key string
}

type ChatStartedMessage struct {
	Room string
}

type ChatMessage struct {
	Room string
	Msg  string
	Name string
}

type RoomOpenedMessage struct {
	Room string
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type ConnectHandler struct {
	Backend Backend
}

type Backend interface {
	AddRoom(room string) error
	AddMessage(room string, msg string, name string) error
}

type RedisBackend struct {
	Redis redis.Conn
}

func (backend *RedisBackend) AddRoom(room string) error {
	backend.Redis.Do("SADD", "rooms", room)
	return nil
}

func (backend *RedisBackend) AddMessage(room string, msg string, name string) error {
	nextId, err := redis.Int64(backend.Redis.Do("INCR", "message:id"))
	if err != nil {
		return err
	}
	msgId := fmt.Sprintf("message:%d", nextId)
	backend.Redis.Do("HMSET", msgId, "author", name, "room", room, "text", msg, "time", time.Now())
	backend.Redis.Do("RPUSH", room+":messages", msgId)
	return nil
}

func (handler *ConnectHandler) handleConnect(so socketio.Socket) {
	log.Println("Handler connect")

	so.On("admin", func(msg AdminMessage) {
		log.Printf("admin: %s\n", msg.Key)
		names[so.Id()] = "Admin"
		so.Join("admins")
		so.Emit("admin_created")
	})

	so.On("chat_started", func(msg ChatStartedMessage) {
		names[so.Id()] = "User"
		roomId := msg.Room
		if msg.Room == "" {
			key := RandStringRunes(8)
			roomId = "room:" + key
		}
		m := make(map[string]interface{})
		m["Room"] = roomId
		handler.Backend.AddRoom(roomId)
		so.Join(roomId)
		so.BroadcastTo("admins", "new_room", m)
		so.Emit("new_room", m)
		log.Printf("new_room: %s\n", roomId)
	})

	so.On("room_opened", func(msg RoomOpenedMessage) {
		log.Printf("room_opened: %s\n", msg.Room)
		so.Join(msg.Room)
	})

	so.On("message", func(msg ChatMessage) {
		log.Printf("Message received for room %s: %s\n",
			msg.Room, msg.Msg)
		msg.Name = names[so.Id()]
		handler.Backend.AddMessage(msg.Room, msg.Name, msg.Msg)
		so.BroadcastTo(msg.Room, "message", msg)
		so.Emit("message", msg)
	})
}

var names map[string]string

func main() {
	conn, err := redis.Dial("tcp", ":7777")
	if err != nil {
		// handle error
	}
	defer conn.Close()

	names = make(map[string]string)
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	connectHandler := &ConnectHandler{&RedisBackend{conn}}
	server.On("connection", func(so socketio.Socket) { connectHandler.handleConnect(so) })
	server.On("reconnection", func(so socketio.Socket) { connectHandler.handleConnect(so) })
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
