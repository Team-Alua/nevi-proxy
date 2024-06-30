package main

import (
	"net/http"
    "fmt"
	"encoding/binary"

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

var clientList *clients.List

func HandleMessages() {
    mailer := clientList.Mailer
    for {
        msg := <-mailer
        id := binary.BigEndian.Uint64(msg[0:8])
        client := clientList.GetClient(id)
        // Client is invalid so ignore
        if client == nil {
            continue
        }

        if len(msg) < 16 {
            data := make([]byte, 16)
            binary.BigEndian.PutUint64(data, uint64(0))
            binary.BigEndian.PutUint64(data[8:], ^uint64(0))
            client.GetWriter().Write(clients.NewBinaryMessage(data))
            continue
        }
        code := msg[8:16]
        fmt.Println(id, msg, code)
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
    clientList = clients.NewList()
	go clientList.PingClients()
    go HandleMessages()
	http.HandleFunc("/nevi-proxy", NeviProxy)
	http.ListenAndServe(":8080", nil)
	return
}

