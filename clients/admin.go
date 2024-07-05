package clients

import (
    "github.com/Team-Alua/nevi-proxy/constants"
)

type Admin struct {
    list *List
}

func NewAdmin() *Admin {
    return &Admin{}
}

func (a *Admin) SendTo(id uint64, m *Mail) {
    list := a.list
    c := list.GetClient(id)
    if c == nil {
        return
    }
    c.SendMail(m.ToBytes())
    // It will send whatever mail is to the client with id
}

func (a *Admin) Forward(m *Mail) {
    // it will forward the message to m.target
    a.SendTo(m.GetTarget(), m)
}

func (a *Admin) HandleDisconnect(id uint64, friendIds [32]uint64) {
    if a == nil {
        return
    }
    for _, friendId := range friendIds {
        if friendId == 0 {
            continue
        }
        m := NewMail(id, friendId, constants.REMOVE_FRIEND, nil)
        a.Forward(m)
    }
}

