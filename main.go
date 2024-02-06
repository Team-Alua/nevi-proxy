package main

import (

    "fmt"
    "net/http"
	"encoding/json"

    "github.com/gorilla/websocket"
)
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

type Client struct {
	As string `json:"as"`
	Filter string `json:"filter"`
	Conn websocket.Conn
	Available bool
	Id string
}

var clientChan chan *Client

func matchMaker(client *Client, potentials []*Client) *Client {
	if client == nil {
		return
	}

	for potential := range potentials {
		if potential == nil {
			continue
		}

		if !potential.Available {
			continue
		}
		if potential.Filter == client.Filter {
			return potential
		}
	}

	return nil
}

func clientTracker() {

	clients := make([]*Client, 0)
	servers := make([]*Client, 0)
	for {
		select {
		case c := <-clientChan:
			if c == nil {
				continue
			}
			var client *Client
			var server *Client
			if c.As == "server" {
				client = matchMaker(c, clients)
				server = c
				servers = append(servers, c)
			} else {
				server = matchMaker(c, servers)
				client = c
				if server == nil {
					clients = append(clients, c)
				}
			}
			if client != nil && server != nil {
				// Go do conversation
			}
		case id := <-endChan:
			// client:id
			// - only removes the client from the list
			// - check if another client needs to be serviced
			// - otherwise, marks server as available
			// all:id
			// - removes both the server and client from the list
		}
	}
}

func NeviProxy(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)

    msgType, msg, err := conn.ReadMessage()
    if err != nil {
        return
    }
	var c Client	
	if err := json.Unmarshal(msg, &c); err != nil {
		conn.Close()
		return
	}
	c.Conn = conn
	clientChan <- &c
})


func main() {
    http.HandleFunc("/nevi-proxy", NeviProxy)
    http.ListenAndServe(":8080", nil)
}
