package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"

	"github.com/gorilla/websocket"
)

var wsChan = make(chan WsPayload)

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type SafeClients struct {
	mu sync.Mutex
	v  map[WebSocketConnection]string
}

// We build this function to protect concurrency access to our map
func (sc *SafeClients) Value(key WebSocketConnection) string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.v[key]
}

// We build this function to protect concurrency access to our map
func (su *SafeClients) Add(key WebSocketConnection) {
	su.mu.Lock()
	su.v[key] = ""
	su.mu.Unlock()
}

var clients = SafeClients{v: make(map[WebSocketConnection]string)}

type WebSocketConnection struct {
	*websocket.Conn
}

// WsJsonResponse defines the response sent back from websocket
type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	Userrole       string   `json: "userrole"`
	Etype          string   `json: "etype"`
	Payload        any      `json: "payload"`
	ConnectedUsers []string `json: "connect_users"`
}

type WsPayload struct {
	Action   string              `json: "action"`
	Username string              `json: "username"`
	Userrole string              `json: "userrole"`
	Message  string              `json: "message"`
	Etype    string              `json: "etype"`
	Payload  any                 `json: "payload"`
	Conn     WebSocketConnection `json: "-"`
}

// WsEndpoint upgrades
func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Client connected to endpoint")

	var response WsJsonResponse
	response.Message = `<em><small>Connected to server</small></em>`

	conn := WebSocketConnection{Conn: ws}
	clients.Add(conn)

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}

	go ListenForWs(&conn)

}

func ListenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload

	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			log.Print(err)
			conn.Close()
			break
		} else {

			log.Print(payload)
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

func broadcastToAll(response WsJsonResponse) {
	for client := range clients.v {
		err := client.WriteJSON(response)
		if err != nil {
			_ = client.Close()
			delete(clients.v, client)
		}

	}
}

func ListenToWsChannel() {
	var response WsJsonResponse

	for {
		e := <-wsChan

		switch e.Action {
		case "user":
			clients.Add(e.Conn) // = e.Userrole
		case "message":
			//users := getUserList()
			response.Action = "message"
			response.Message = e.Message
			//response.ConnectedUsers = users
			response.Etype = e.Etype
			response.Userrole = e.Userrole
			response.Payload = e.Payload
			broadcastToAll(response)

		case "left":
			response.Action = "user_left"
			delete(clients.v, e.Conn)
			e.Conn.Close()
			users := getUserList()
			response.ConnectedUsers = users
			broadcastToAll(response)
		}
	}
}

func getUserList() []string {
	var userList []string
	for _, x := range clients.v {
		userList = append(userList, x)
	}
	sort.Strings(userList)
	return userList
}
