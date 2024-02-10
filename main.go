package main

import (

    "net/http"
	"encoding/json"

    "github.com/gorilla/websocket"
)
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clientChan chan *Client
var serverChan chan *Server

func clientTracker() {
	matcher := NewMatcher()
	remover := make(chan *Server)
	for {
		select {
		case c := <-clientChan:
			// Only repeat so we can send notifications
			go c.Repeat()
			matcher.AddClient(c)
			matcher.MatchClients()
		case s := <-serverChan:
			s.Remover = remover
			// Setup everything so everyone can start talking to each other
			go s.Listen()
			go s.Repeat()
			go s.Ping()
			matcher.AddServer(s)
			matcher.MatchClients()
		case s := <-remover:
			matcher.RemoveServer(s)
			s.Close()
		}
	}
}

type SortRequest struct {
	As string `json:"as"`
	Filter string `json:"filter"`
	Limit uint32 `json:"limit,omitempty"`
}

func NeviProxy(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
	
	if err != nil {
		panic(err)
	}

    _, msg, err := conn.ReadMessage()
    if err != nil {
        return
    }

	var sr SortRequest
	if err := json.Unmarshal(msg, &sr); err != nil {
		conn.Close()
		return
	}

	// TODO: Limit the amount of clients a server can handle
	// TODO: Automatically disconnect if max amount of servers + clients is met
	if sr.As == "server" && sr.Limit > 0 {
		serverChan <- NewServer(conn, sr.Filter, sr.Limit)
	} else if sr.As == "client" {
		clientChan <- NewClient(conn, sr.Filter)
	} else {
		conn.Close()
	}
}


func main() {
	serverChan = make(chan *Server)
	clientChan = make(chan *Client)
	go clientTracker()
    http.HandleFunc("/nevi-proxy", NeviProxy)
    http.ListenAndServe(":8080", nil)
	return
}

