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

type RoomState struct {
	PlayerList    []string `json:"playerList"`
	AuditorList   []string `json:"auditorList"`
	Tick          int      `json:"tick"`
	CurrentPlayer string   `json:"currentPlayer"`
	TimeToExplode float32  `json:"timeToExplode"`
}

type Person struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type EnterEvent struct {
	Type      string    `json:"type"`
	RoomState RoomState `json:"roomState"`
	Who       Person    `json:"who"`
}

type GameOnEvent struct {
	Type      string    `json:"type"`
	RoomState RoomState `json:"roomState"`
}

type NextTurnEvent struct {
	Type      string    `json:"type"`
	RoomState RoomState `json:"roomState"`
}

type TickEvent struct {
	Type      string    `json:"type"`
	RoomState RoomState `json:"roomState"`
	Tick      int       `json:"tick"`
}

type LeaveEvent struct {
	Type      string    `json:"type"`
	RoomState RoomState `json:"roomState"`
	Who       Person    `json:"who"`
}

type LoseAndNextRoundEvent struct {
	Type      string    `json:"type"`
	RoomState RoomState `json:"roomState"`
	Who       Person    `json:"who"`
}

type LoseAndGameOverEvent struct {
	Type      string    `json:"type"`
	RoomState RoomState `json:"roomState"`
	Who       Person    `json:"who"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type client struct {
	Ch     chan string
	Player Person
}

type Action struct {
	Time float32
	Who  Person
}

type Room struct {
	Entering chan client
	Leaving  chan client
	Actions  chan Action
	Clients  map[client]bool
	Seats    int
	Begin    bool
	Over     bool
	State    RoomState
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
		Ch: ch,
		Player: Person{
			Name:   RandStringRunes(8),
			Avatar: "http://xxx.xxx.xxx/xxxxxx",
		},
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

func (room *Room) begin() {
	room.State.CurrentPlayer = room.State.PlayerList[0]
	room.State.TimeToExplode = 5
	room.State.Tick = 10

	room.Begin = true
}

func (room *Room) lose(name string) {
	for i, val := range room.State.PlayerList {
		if val == name {
			room.State.PlayerList = append(room.State.PlayerList[:i], room.State.PlayerList[i+1:]...)
			room.State.AuditorList = append(room.State.AuditorList, name)
		}
	}

	i := rand.Int() % len(room.State.PlayerList)
	room.State.CurrentPlayer = room.State.PlayerList[i]
	room.State.Tick = 10
	room.State.TimeToExplode = 5
}

func (room *Room) nextTurn() {
	for i, val := range room.State.PlayerList {
		if val == room.State.CurrentPlayer {
			room.State.CurrentPlayer = room.State.PlayerList[(i+1)%len(room.State.PlayerList)]
		}
	}

	room.State.Tick = 10
}

func (room *Room) broadcast(msg string) {
	fmt.Println("broadcast:" + msg)
	for cli := range room.Clients {
		cli.Ch <- msg
	}
}

func (room *Room) leave(cli client) {
	delete(room.Clients, cli)
	close(cli.Ch)
	for i, val := range room.State.PlayerList {
		if val == cli.Player.Name {
			room.State.PlayerList = append(room.State.PlayerList[:i], room.State.PlayerList[i+1:]...)
		}
	}
	for i, val := range room.State.AuditorList {
		if val == cli.Player.Name {
			room.State.AuditorList = append(room.State.AuditorList[:i], room.State.AuditorList[i+1:]...)
		}
	}
}

func main() {
	tick := time.Tick(1000 * time.Millisecond)
	room := Room{}
	room.Entering = make(chan client)
	room.Leaving = make(chan client)
	room.Actions = make(chan Action)
	room.Clients = make(map[client]bool)
	room.Seats = 3
	Rooms["666666"] = room

	go func(room *Room) {
		for {
			select {
			case <-tick:
				if room.Begin && !room.Over {
					room.State.Tick -= 1
					if room.State.Tick <= 0 {
						go func() {
							room.Actions <- Action{
								Time: 100.0,
								Who: Person{
									Name:   room.State.CurrentPlayer,
									Avatar: "",
								},
							}
						}()
					} else if room.State.Tick <= 3 {
						event := TickEvent{
							Type:      "tick",
							Tick:      room.State.Tick,
							RoomState: room.State,
						}
						jsonBytes, _ := json.Marshal(event)
						room.broadcast(string(jsonBytes))
					}
				}

			case action := <-room.Actions:
				room.State.TimeToExplode -= action.Time

				if room.State.TimeToExplode <= 0 {
					room.lose(action.Who.Name)
					if len(room.State.PlayerList) <= 1 {
						room.Over = true
						event := LoseAndGameOverEvent{
							Type:      "loseAndGameOver",
							Who:       action.Who,
							RoomState: room.State,
						}
						jsonBytes, _ := json.Marshal(event)
						room.broadcast(string(jsonBytes))
					} else {
						event := LoseAndNextRoundEvent{
							Type:      "loseAndNextRound",
							Who:       action.Who,
							RoomState: room.State,
						}
						jsonBytes, _ := json.Marshal(event)
						room.broadcast(string(jsonBytes))
					}
				} else {
					room.nextTurn()
					event := NextTurnEvent{
						Type:      "nextTurn",
						RoomState: room.State,
					}
					jsonBytes, _ := json.Marshal(event)
					room.broadcast(string(jsonBytes))
				}

			case cli := <-room.Entering:
				room.Clients[cli] = true
				gameOn := false

				if room.Begin {
					room.State.AuditorList = append(room.State.AuditorList, cli.Player.Name)
				} else {
					room.State.PlayerList = append(room.State.PlayerList, cli.Player.Name)
				}

				if !room.Begin && len(room.State.PlayerList) >= room.Seats {
					room.begin()
					gameOn = true
				}
				event := EnterEvent{
					Type:      "enter",
					Who:       cli.Player,
					RoomState: room.State,
				}
				gameOnEvent := GameOnEvent{
					Type:      "game on",
					RoomState: room.State,
				}
				jsonBytes, _ := json.Marshal(event)
				room.broadcast(string(jsonBytes))
				if gameOn {
					room.nextTurn()
					jsonBytes, _ := json.Marshal(gameOnEvent)
					room.broadcast(string(jsonBytes))
				}

			case cli := <-room.Leaving:

				room.leave(cli)

				event := LeaveEvent{
					Type:      "leave",
					Who:       cli.Player,
					RoomState: room.State,
				}

				jsonBytes, _ := json.Marshal(event)
				room.broadcast(string(jsonBytes))

				if len(room.State.PlayerList) == 0 && len(room.State.AuditorList) == 0 {
					room.Begin = false
					room.Over = false
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
