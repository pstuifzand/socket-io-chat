package main

import (
	//"fmt"
	socketio "github.com/googollee/go-socket.io"
	"log"
	"math/rand"
	"net/http"
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

func handleConnect(so socketio.Socket) {
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
			roomId = "room-" + key
		}
		m := make(map[string]interface{})
		m["Room"] = roomId
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
		so.BroadcastTo(msg.Room, "message", msg)
		so.Emit("message", msg)
	})
}

var names map[string]string

func main() {
	names = make(map[string]string)
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("connection", handleConnect)
	server.On("reconnect", handleConnect)
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
