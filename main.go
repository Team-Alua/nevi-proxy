package main

import (
	"net/http"
    "fmt"

	"github.com/gorilla/websocket"
	"github.com/Team-Alua/nevi-proxy/clients"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clientList *ClientList

func HandleMessages() {
    mailer := clientList.BinaryHandler
    for {
        msg := <-mailer
        fmt.Println(msg)
    }
}

func NeviProxy(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
	
	if err != nil {
		panic(err)
	}

    c := clients.NewClient(conn)
    clientList.AddClient(c)
	go c.Listen()
	go c.Repeat()
}

func main() {
    clientList = NewClientList()
	go clientList.PingClients()
    go HandleMessages()
	http.HandleFunc("/nevi-proxy", NeviProxy)
	http.ListenAndServe(":8080", nil)
	return
}

