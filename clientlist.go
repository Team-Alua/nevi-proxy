package main

import (
    "slices"
    "sync"
    "time"

	"github.com/Team-Alua/nevi-proxy/isync"
	"github.com/Team-Alua/nevi-proxy/clients"
)

type ClientList struct {
    BinaryHandler chan []byte
	nextId *isync.Incrementer[uint64]
	lock sync.RWMutex
    clients []*clients.Client
    ids []uint64
}

func NewClientList() *ClientList {
    cl := &ClientList{}
    cl.BinaryHandler = make(chan []byte, 10)
    cl.nextId = isync.NewIncrementer[uint64](1)

    cl.clients = make([]*clients.Client, 0)
    cl.ids = make([]uint64, 0)
    return cl
}

func (cl *ClientList) AddClient(c *clients.Client) {
    id := cl.nextId.Increment()
    c.Id = id
    c.BinaryHandler = cl.BinaryHandler
    l := cl.lock
    l.Lock()
    cl.clients = append(cl.clients, c)
    cl.ids = append(cl.ids, id)
    l.Unlock()
}

func (cl *ClientList) RemoveClient(id uint64) {
    if id == 0 {
        return
    }
    l := cl.lock
    l.Lock()
    idx, found := slices.BinarySearch(cl.ids, id)
    if found {
        // Remove the reference to the client
        cl.clients[idx] = nil
        clen := len(cl.clients)
        // Shift all the clients to the left by one
        for i := idx + 1; i < clen; i++ {
            cl.clients[i - 1] = cl.clients[i]
            cl.ids[i - 1] = cl.ids[i]
        }
        // Nil and shrink slice
        cl.clients[clen - 1] = nil
        cl.clients = cl.clients[:clen - 1]
        cl.ids[clen - 1] = 0
        cl.ids = cl.ids[:clen - 1]
    }
    l.Unlock()
}

func (cl *ClientList) GetClient(id uint64) (c *clients.Client) {
    if id == 0 {
        return nil
    }
    l := cl.lock
    l.Lock()
    idx, found := slices.BinarySearch(cl.ids, id)
    if found {
        c = cl.clients[idx]
    }
    l.Unlock()
    return
}

func (cl *ClientList) PingClients() {
	pingPeriod := 1 * time.Second
	t1 := time.NewTicker(pingPeriod)
	defer func() {
		t1.Stop()
	}()

	for { 
		<-t1.C
        l := cl.lock
        l.Lock()

        disc := make([]uint64, 0)
		ping := clients.NewPingMessage()
		for idx, client := range cl.clients {
			if !client.IsConnected() {
				disc = append(disc, cl.ids[idx])
				continue
			}
			client.GetWriter().Write(ping)
		}

        l.Unlock()

        for _, id := range disc {
            cl.RemoveClient(id)
        }
	}
}
