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

type Client struct {
	As string `json:"as"`
	Filter string `json:"filter"`
	Conn *websocket.Conn
	Available bool
	Id string
}

var clientChan chan *Client


func RemoveClient(clients []*Client, client *Client) []*Client {
	ret := clients
	i := -1
	for idx, c := range clients {
		if client == c {
			i = idx
		}
	}
	if i >= 0 {
		ret = make([]*Client, 0)
		ret = append(ret, clients[:i]...)
		ret = append(ret, clients[i+1:]...)
	}
	return ret
}
func matchMaker(potentials []*Client, client *Client) *Client {
	if client == nil {
		return nil
	}

	for _, potential := range potentials {
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


func findServerById(servers []*Client, id string) *Client {
	for _, server := range servers {
		if server == nil {
			continue
		}
		if server.Id == id {
			return server
		}
	}
	return nil
}

func clientTracker() {

	clients := make([]*Client, 0)
	servers := make([]*Client, 0)
	endChan := make(chan string)
	for {
		var client *Client
		var server *Client

		select {
		case c := <-clientChan:
			if c == nil {
				continue
			}
			// Any client here is already available
			if c.As == "server" {
				server = c
				client = matchMaker(clients, server)
				if client != nil {
					// Remove client from list
					clients = RemoveClient(clients, client)
				}
				servers = append(servers, server)
			} else if c.As == "client" {
				client = c
				server = matchMaker(servers, client)
				if server == nil {
					clients = append(clients, client)
				}
			}
		case id := <-endChan:
			rm := id[0] == '!'
			if rm {
				id = id[1:]
			}
			server := findServerById(servers, id)
			if rm || server == nil {
				servers = RemoveClient(servers, server)
				server = nil
			} else {
				client = matchMaker(clients, server)
				if client != nil {
					// Client found
					// Remove client from list
					clients = RemoveClient(clients, client)
				} else {
					server.Available = true
				}
			}
		}

		if client != nil && server != nil {
			client.Available = false
			server.Available = false
			// Go do communication here
		}
	}
}

func NeviProxy(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)

    _, msg, err := conn.ReadMessage()
    if err != nil {
        return
    }
	var c Client	
	if err := json.Unmarshal(msg, &c); err != nil {
		conn.Close()
		return
	}
	c.Conn = conn
	c.Available = true
	clientChan <- &c
}


func main() {
    http.HandleFunc("/nevi-proxy", NeviProxy)
    http.ListenAndServe(":8080", nil)
	return
}
