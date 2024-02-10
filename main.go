package main

import (

    "net/http"
	"encoding/json"

    "github.com/gorilla/websocket"
)
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

var clientChan chan *Client
var serverChan chan *Server

func clientTracker() {
	matcher := NewMatcher()
	for {
		select {
		case c := <-clientChan:
			matcher.AddClient(c)
			matcher.MatchClients()
		case s := <-serverChan:
			matcher.AddServer(s)
			matcher.MatchClients()
		}
	}
}

type SortRequest struct {
	As string `json:"as"`
	Filter string `json:"filter"`
	Limit uint64 `json:"limit,omitempty"`
}

func NeviProxy(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)

    _, msg, err := conn.ReadMessage()
    if err != nil {
        return
    }

	var sr SortRequest
	if err := json.Unmarshal(msg, &sr); err != nil {
		conn.Close()
		return
	}

	if sr.As == "server" {
		var serv Server	
		serv.Conn = conn
		serv.Filter = sr.Filter
		serv.Limit = sr.Limit
		serv.Clients = make([]*Client, 0)
		serverChan <- &serv
	} else if sr.As == "client" {
		var client Client
		client.Conn = conn
		client.Filter = sr.Filter
		clientChan <- &client
	} else {
		conn.Close()
	}
}


func main() {
	go clientTracker()
    http.HandleFunc("/nevi-proxy", NeviProxy)
    http.ListenAndServe(":8080", nil)
	return
}

