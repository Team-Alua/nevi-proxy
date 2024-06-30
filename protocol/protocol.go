package protocol

import (
    "github.com/Team-Alua/nevi-proxy/clients"
)

type Protocol struct {
    mailer chan []byte
    clientList *clients.List
}

func NewProtocol(mailer chan []byte, clientList *clients.List) *Protocol {
    return &Protocol{mailer, clientList}
}

