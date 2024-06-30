package main

import (
    "net/http"

    "github.com/gorilla/websocket"
    "github.com/Team-Alua/nevi-proxy/clients"
    "github.com/Team-Alua/nevi-proxy/protocol"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

var clientList *clients.List

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
    proto := protocol.NewProtocol(clientList.Mailer, clientList)
    go proto.HandleMail()
    http.HandleFunc("/nevi-proxy", NeviProxy)
    http.ListenAndServe(":8080", nil)
    return
}

