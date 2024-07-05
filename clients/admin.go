package clients

import (
    "github.com/Team-Alua/nevi-proxy/constants"
)

type Admin struct {
    list *List
}

func NewAdmin(list *List) *Admin {
    return &Admin{list}
}

func (a *Admin) SendTo(id uint64, m *Mail) {
    list := a.list
    c := list.GetClient(id)
    if c == nil {
        return
    }
    c.SendMail(m.ToBytes())
}

func (a *Admin) Forward(m *Mail) {
    a.SendTo(m.GetTarget(), m)
}

func (a *Admin) HandleDisconnect(id uint64) {
    if a == nil {
        return
    }
    c := a.list.GetClient(id)
    for _, friendId := range c.GetFriends() {
        if friendId == 0 {
            continue
        }
        m := NewMail(id, friendId, constants.REMOVE_FRIEND, nil)
        a.Forward(m)
    }
}

