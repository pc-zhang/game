package main

import (
	"net/http"

	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"math/rand"
	"time"
)

type EventT struct {
	PlayerList    []string `json:"playerList"`
	AuditorList   []string `json:"auditorList"`
	Tick          int      `json:"tick"`
	CurrentPlayer string   `json:"currentPlayer"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type client struct {
	Ch       chan string
	UserName string
}

type Room struct {
	Entering chan client
	Leaving  chan client
	Messages chan string
	Clients  map[client]bool
}

var Rooms = make(map[string]Room)
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	_, p, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}
	roomNum := string(p)

	ch := make(chan string)
	cli := client{
		Ch:       ch,
		UserName: RandStringRunes(8),
	}

	room, ok := Rooms[roomNum]
	if !ok {
		conn.WriteJSON("wrong room number!")
		conn.Close()
		return
	}

	room.Entering <- cli

	go connWriter(conn, cli.Ch)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			room.Leaving <- cli
			conn.Close()
			return
		}
	}
}

func connWriter(conn *websocket.Conn, clientChan <-chan string) {
	for msg := range clientChan {
		conn.WriteJSON(msg)
	}
}

func main() {
	room := Room{}
	room.Entering = make(chan client)
	room.Leaving = make(chan client)
	room.Messages = make(chan string)
	room.Clients = make(map[client]bool)
	Rooms["666666"] = room

	go func(room *Room) {
		for {
			select {
			case msg := <-room.Messages:
				for cli := range room.Clients {
					cli.Ch <- msg
				}
			case cli := <-room.Entering:
				room.Clients[cli] = true
				event := EventT{}
				event.PlayerList = make([]string, 0)
				for cli := range room.Clients {
					event.PlayerList = append(event.PlayerList, cli.UserName)
				}

				for cli := range room.Clients {
					jsonBytes, _ := json.Marshal(event)
					cli.Ch <- string(jsonBytes)
				}
			case cli := <-room.Leaving:
				delete(room.Clients, cli)
				close(cli.Ch)

				event := EventT{}
				event.PlayerList = make([]string, 0)
				for cli := range room.Clients {
					event.PlayerList = append(event.PlayerList, cli.UserName)
				}

				for cli := range room.Clients {
					jsonBytes, _ := json.Marshal(event)
					cli.Ch <- string(jsonBytes)
				}
			}
		}
	}(&room)

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.GET("/ws", func(c *gin.Context) {
		wshandler(c.Writer, c.Request)
	})

	router.Run(":8000")
}
