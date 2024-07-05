package clients

import (
    "slices"
    "sync"
    "time"

    "github.com/Team-Alua/nevi-proxy/isync"
)

type List struct {
    Mailer chan []byte
    nextId *isync.Incrementer[uint64]
    lock sync.RWMutex
    admin *Admin
    clients []*Client
    ids []uint64
}

func NewList() *List {
    cl := &List{}
    cl.Mailer = make(chan []byte, 10)
    cl.nextId = isync.NewIncrementer[uint64](1)

    cl.admin = NewAdmin(cl)
    cl.clients = make([]*Client, 0)
    cl.ids = make([]uint64, 0)
    return cl
}

func (cl *List) GetAdmin() *Admin {
    if cl == nil {
        return nil
    }
    return cl.admin
}

func (cl *List) AddClient(c *Client) {
    id := cl.nextId.Increment()
    c.SetId(id)
    c.SetMailer(cl.Mailer)

    l := cl.lock
    l.Lock()
    cl.clients = append(cl.clients, c)
    cl.ids = append(cl.ids, id)
    l.Unlock()
}

func (cl *List) RemoveClient(id uint64) {
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


func (cl *List) GetClient(id uint64) (c *Client) {
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

func (cl *List) GetClientsWithTag(tag uint64) (d []uint64) {
    d = make([]uint64, 0)
    // This is the no tag constant
    if tag == 0 {
        return
    }

    l := cl.lock
    l.RLock()
    for idx, client := range cl.clients {
        if client.GetTag() == tag {
            d = append(d, cl.ids[idx])
        }
    }
    l.RUnlock()
    return
}

func (cl *List) PingClients() {
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
        ping := NewPingMessage()
        for idx, client := range cl.clients {
            if !client.IsConnected() {
                disc = append(disc, cl.ids[idx])
                continue
            }
            client.GetWriter().Write(ping)
        }

        l.Unlock()
        for _, id := range disc {
            cl.admin.HandleDisconnect(id)
            cl.RemoveClient(id)
        }
    }
}


