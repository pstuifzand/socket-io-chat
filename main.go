package main

import (
	"encoding/json"
	socketio "github.com/googollee/go-socket.io"
	"log"
	"math/rand"
	"net/http"
	"regexp"
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

type ApiHandler struct {
	Backend Backend
}

type RoomsResponse struct {
	Rooms []Room
}

type RoomResponse struct {
	Room     Room
	Messages []Message
}

var roomUrlRegex *regexp.Regexp

func (this *ApiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// admin api /api/rooms
	if r.URL.Path == "/api/rooms" {
		rooms, _ := this.Backend.GetRooms()
		for i, room := range rooms {
			if abs, err := r.URL.Parse(room.Href); err == nil {
				abs.Scheme = "http"
				abs.Host = r.Host
				rooms[i].Href = abs.String()
			} else {
				log.Fatal(err)
			}
		}
		resp := &RoomsResponse{rooms}
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	} else if matches := roomUrlRegex.FindStringSubmatch(r.URL.Path); matches != nil {
		roomId := matches[1]
		_, messages, _ := this.Backend.GetRoom(roomId)
		resp := &RoomResponse{Room{roomId, ""}, messages}
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	http.NotFound(w, r)
}

type ConnectHandler struct {
	Backend Backend
	Names   map[string]string
}

func (handler *ConnectHandler) handleConnect(so socketio.Socket) {
	log.Println("Handler connect")

	so.On("admin", func(msg AdminMessage) {
		log.Printf("admin: %s\n", msg.Key)
		handler.Names[so.Id()] = "Admin"
		so.Join("admins")
		so.Emit("admin_created")
	})

	so.On("chat_started", func(msg ChatStartedMessage) {
		handler.Names[so.Id()] = "User"
		roomId := msg.Room
		if msg.Room == "" {
			key := RandStringRunes(8)
			roomId = "room-" + key
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
		msg.Name = handler.Names[so.Id()]
		handler.Backend.AddMessage(msg.Room, msg.Name, msg.Msg)
		so.BroadcastTo(msg.Room, "message", msg)
		so.Emit("message", msg)
	})
}

func NewConnectHandler(backend Backend) *ConnectHandler {
	return &ConnectHandler{backend, make(map[string]string)}
}

func main() {
	rand.Seed(time.Now().Unix())
	roomUrlRegex = regexp.MustCompile("^/api/rooms/(room-\\w+)$")

	backend, _ := NewRedisBackend("redis:6379")
	defer backend.Close()

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	connectHandler := NewConnectHandler(backend)
	server.On("connection", func(so socketio.Socket) { connectHandler.handleConnect(so) })
	server.On("reconnection", func(so socketio.Socket) { connectHandler.handleConnect(so) })
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/api/", &ApiHandler{backend})
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
