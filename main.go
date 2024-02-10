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
	Limit uint32 `json:"limit,omitempty"`
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
		serverChan <- NewServer(conn, sr.Filter, sr.Limit)
	} else if sr.As == "client" {
		clientChan <- NewClient(conn, sr.Filter)
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

