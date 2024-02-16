package main

import (

	"net/http"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/matcher"
	"github.com/Team-Alua/nevi-proxy/clients"
	"github.com/Team-Alua/nevi-proxy/isync"

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
var pingChan chan Connectable

type Connectable interface {
	IsConnected() bool
	Close() 
	GetWriter() *isync.ReadWriter[*clients.Message]
}


func pinger() {
	pingable := make([]Connectable, 0)

	pingPeriod := 1 * time.Second
	t1 := time.NewTicker(pingPeriod)
	defer func() {
		t1.Stop()
	}()

	for { 
		select {
		case c := <-pingChan:
			pingable = append(pingable, c)
		case <-t1.C:
			disconnected := false
			ping := clients.NewPingMessage()
			for _, client := range pingable {
				if client.IsConnected() {
					disconnected = true
					continue
				}
				client.GetWriter().Write(ping)
			}
	
			if disconnected {
				temp := make([]Connectable, 0)
				for _, client := range pingable {
					if client.IsConnected() {
						temp = append(temp, client)
					}
				}
				pingable = temp
			}
		}
	}
}

func clientTracker() {
	match := matcher.New()
	remover := make(chan *clients.Server)
	rematch := make(chan *clients.Server)
	var clientId uint64

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
			c.Server.Set(nil)
			go c.Listen()
			go c.Repeat()
			match.MatchClient(c)
			pingChan <- c
		case s := <-serverChan:
			s.Remover = remover
			s.Rematch = rematch
			// Setup everything so everyone can start talking to each other
			go s.Listen()
			go s.Repeat()
			go s.Ping()
			match.Servers.Add(s)
			match.MatchServer(s)
			pingChan <- s
		case s := <-remover:
			match.Servers.Remove(s)
			s.Close()
		}
	}
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


	if cType == SERVER {
		var r clients.ServerRequest
		err := json.Unmarshal(msg, &r)
		sc := false
		if err != nil {
			sc = true
		} else if r.Limit == 0 {
			sc = true
		} else if 64 < r.Limit {
			sc = true
		}

		if sc {
			conn.Close()
		} else {
			serverChan <- clients.NewServer(conn, &r)
		}
	} else {
		var r clients.ClientRequest
		err := json.Unmarshal(msg, &r)
		if err != nil {
			conn.Close()
		} else {
			clientChan <- clients.NewClient(conn, &r)
		}
	}
}

func main() {
	serverChan = make(chan *clients.Server)
	clientChan = make(chan *clients.Client)
	pingChan = make(chan Connectable)
	go pinger()
	go clientTracker()
	http.HandleFunc("/nevi-proxy", NeviProxy)
	http.ListenAndServe(":8080", nil)
	return
}

