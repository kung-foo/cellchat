package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/tideland/golib/cells"
	"github.com/tideland/golib/identifier"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var roomTmpl = template.Must(template.ParseFiles("resources/room.html"))

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type webSocketBehavior struct {
	nullBehavior
	ctx    cells.Context
	ws     *websocket.Conn
	user   string
	output chan []byte
}

func newWebSocketBehavior(ws *websocket.Conn, user string) *webSocketBehavior {
	wsb := &webSocketBehavior{
		ws:     ws,
		user:   user,
		output: make(chan []byte, 32),
	}
	return wsb
}

func (wsb *webSocketBehavior) Init(ctx cells.Context) (err error) {
	wsb.ctx = ctx
	return
}

func (wsb *webSocketBehavior) sendUserLeaveEvent() {
	wsb.ctx.Environment().Request(makeUserID(wsb.user), USER_DISCONNECTED, nil, nil, time.Second*1)
}

func (wsb *webSocketBehavior) ProcessEvent(event cells.Event) (err error) {
	switch event.Topic() {
	case SAYS_TO:
		to, ok := event.Payload().Get("to")
		if !ok || to.(string) != makeUserID(wsb.user) {
			return
		}
		msg, err := json.Marshal(payloadToValues(event.Payload()))
		if err != nil {
			log.Fatal(err)
		}
		wsb.output <- msg
	case SAYS_ALL:
		msg, err := json.Marshal(payloadToValues(event.Payload()))
		if err != nil {
			log.Fatal(err)
		}
		wsb.output <- msg
	}
	return
}

func (wsb *webSocketBehavior) readPump() {
	defer func() {
		wsb.ws.Close()
	}()
	wsb.ws.SetReadLimit(maxMessageSize)
	wsb.ws.SetReadDeadline(time.Now().Add(pongWait))
	wsb.ws.SetPongHandler(func(string) error { wsb.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := wsb.ws.ReadMessage()
		if err != nil {
			if err != io.EOF {
				log.Error(err, message)
			}
			break
		}
		rootEnvironment.EmitNew(makeUserID(wsb.user), SAY, cells.PayloadValues{
			"room":    makeRoomID("cafeteria", "school"),
			"message": string(message),
		}, nil)
	}
}

// write writes a message with the given message type and payload.
func (wsb *webSocketBehavior) write(mt int, payload []byte) error {
	wsb.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return wsb.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (wsb *webSocketBehavior) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		wsb.ws.Close()
	}()
	for {
		select {
		case msg, ok := <-wsb.output:
			if !ok {
				wsb.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := wsb.write(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := wsb.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func listRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms, _ := rootEnvironment.Request(makeBuildingID("school"), LIST_ROOMS, nil, nil, time.Second*1)

	for _, room := range rooms.([]string) {
		tokens := strings.Split(room, ":")
		fmt.Fprintf(w, "<div><a href=\"/%s/%s\">%s</a></div>\n", tokens[1], tokens[2], room)
	}
}

func roomHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	building := vars["building"]
	room := vars["room"]

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	p := map[string]string{
		"host":     r.Host,
		"room":     room,
		"building": building,
	}
	roomTmpl.Execute(w, p)
}

func wsRoomHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")

	if user == "" || user == "undefined" {
		http.Error(w, "user name not specified", http.StatusTeapot)
		return
	}

	vars := mux.Vars(r)
	building := vars["building"]
	room := vars["room"]
	roomID := makeRoomID(room, building)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	wsb := newWebSocketBehavior(ws, user)
	id := identifier.Identifier("wsb", user, identifier.NewUUID())
	rootEnvironment.StartCell(id, wsb)
	subscribe(rootEnvironment, roomID, id)

	userID := addUser(rootEnvironment, building, room, user)

	defer func() {
		wsb.sendUserLeaveEvent()
		rootEnvironment.StopCell(userID)
		rootEnvironment.StopCell(id)
	}()

	go wsb.writePump()
	wsb.readPump()
}

func startServer(env cells.Environment) {
	r := mux.NewRouter()
	r.HandleFunc("/", listRoomsHandler)
	r.HandleFunc("/ws/{building}/{room}", wsRoomHandler)
	r.HandleFunc("/{building}/{room}", roomHandler)

	http.Handle("/", handlers.LoggingHandler(os.Stdout, r))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
