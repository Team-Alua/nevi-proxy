package main

import (

	"net/http"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/matcher"
	"github.com/Team-Alua/nevi-proxy/clients"
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clientChan chan *clients.Client
var serverChan chan *clients.Server

func clientTracker() {
	match := matcher.New()
	remover := make(chan *clients.Server)
	rematch := make(chan *clients.Server)
	var clientId uint32

	for {
		select {
		case s := <-rematch:
			// Prevent clients from staying in the 
			// queue forever if no more clients/servers
			// connect
			match.MatchServer(s)
		case c := <-clientChan:
			c.Id.Set(clientId)
			clientId += 1
			// Only repeat so we can send notifications
			// Create a dummy read writer
			c.ServerWriter.Set(nil)
			go c.Listen()
			go c.Repeat()
			match.MatchClient(c)
		case s := <-serverChan:
			s.Remover = remover
			s.Rematch = rematch
			// Setup everything so everyone can start talking to each other
			go s.Listen()
			go s.Repeat()
			go s.Ping()
			match.Servers.Add(s)
			match.MatchServer(s)
		case s := <-remover:
			match.Servers.Remove(s)
			s.Close()
		}
	}
}

type SortRequest struct {
	Filter string `json:"filter"`
	Limit uint32 `json:"limit,omitempty"`
}

func NeviProxy(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// Validate token here
	if q.Get("token") == "" {

	}

    conn, err := upgrader.Upgrade(w, r, nil)
	
	if err != nil {
		panic(err)
	}

	clientType := q.Get("type")
	const CLIENT = 0
	const SERVER = 1
	const UNK = 2
	cType := UNK
	if clientType == "client" {
		cType = CLIENT
	} else if clientType == "server" {
		cType = SERVER
	}

	if cType == UNK {
		conn.Close()
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}
	conn.SetReadDeadline(time.Time{})

	var sr SortRequest
	if err := json.Unmarshal(msg, &sr); err != nil {
		return
	}

	if cType == CLIENT {
		serverChan <- clients.NewServer(conn, sr.Filter, sr.Limit)
	} else {
		clientChan <- clients.NewClient(conn, sr.Filter)
	}

}


func main() {
	serverChan = make(chan *clients.Server)
	clientChan = make(chan *clients.Client)
	go clientTracker()
	http.HandleFunc("/nevi-proxy", NeviProxy)
	http.ListenAndServe(":8080", nil)
	return
}

