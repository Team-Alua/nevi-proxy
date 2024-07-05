package protocol

import (
    "encoding/binary"
    "github.com/Team-Alua/nevi-proxy/clients"

)


const (
    SEND_MESSAGE = 1
    UPDATE_STATE = 2
    GET_SERVING_WITH_TAG = 3
    ADD_FRIEND = 4
    REMOVE_FRIEND = 5
)

func (p *Protocol) notifyNone(id uint64, data []byte) {
    c := p.clientList.GetClient(id)
    if c == nil {
        return
    }
    c.SendMail(data)
}

func (p *Protocol) HandleMail() {
    clientList := p.clientList
    admin := clientList.GetAdmin()
    mailer := p.mailer
    for {
        msg := <-mailer
        m := clients.NewMailFromBytes(msg)
        sId := m.GetSource()
        s := clientList.GetClient(sId)
        if s == nil {
            continue
        } 
        switch m.GetCode() {
        case SEND_MESSAGE:
            sId := m.GetSource()
            tId := m.GetTarget()
            t := clientList.GetClient(tId)
            var bounced bool
            // must be a valid target client
            if t == nil {
                bounced = true
            // The tags must match
            } else if s.GetTag() != t.GetTag() {
                bounced = true
            // The target must declare that it is friends
            // with the source to prevent abuse.
            } else if t.FriendsWith(sId) {
                bounced = false
            // if either is not friendly then
            // this message is not sent.
            } else if !s.IsFriendly() || !t.IsFriendly() {
                bounced = true 
            // Not valid interactions:
            // server <=> server 
            // client <=> client
            } else if s.IsServing() == t.IsServing() {
                bounced = true
            }

            if bounced {
                // Mail bounced
                fm := clients.NewMail(0, tId, m.GetCode(), nil)
                admin.SendTo(sId, fm)
            } else {
                // Mail forwarded to target
                fm := clients.NewMail(sId, tId, m.GetCode(), m.GetData())
                admin.Forward(fm)
            }
        case UPDATE_STATE:
            s.SetState(m.GetTarget())
            if s.IsServing() {
                tag := s.GetTag()
                ids := clientList.GetClientsWithTag(tag)
                r := clients.NewMail(0, sId, m.GetCode(), nil)
                for _, id := range ids {
                    n := clientList.GetClient(id)
                    // Ignore servers
                    if n.IsServing() {
                        continue
                    }
                    admin.SendTo(id, r)
                }
            }
            r := clients.NewMail(0, m.GetTarget(), m.GetCode(), nil)
            admin.SendTo(sId, r)
        case GET_SERVING_WITH_TAG: 
            var payload []byte
            // Can not be a server
            if !s.IsServing() {
                ids := clientList.GetClientsWithTag(m.GetTarget())
                payload = make([]byte, 0)
                idx := 0
                for _, id := range ids {
                    n := clientList.GetClient(id)
                    // Ignore clients and ignore servers
                    // who do not want friends
                    if !n.IsServing() || !n.IsFriendly() {
                        continue
                    }
                    // Add 8 bytes
                    payload = append(payload, 0, 0, 0, 0, 0, 0, 0, 0)
                    binary.LittleEndian.PutUint64(payload[8 * idx:], id)
                    idx += 1
                }
            }
            r := clients.NewMail(0, sId, m.GetCode(), payload)
            admin.SendTo(sId, r)
        case ADD_FRIEND:
            tId := m.GetTarget()
            t := clientList.GetClient(tId)
            if t == nil {
                // Do not acknowledge invalid ids
            } else if s.IsServing() == t.IsServing() {
                // Not compatible
            } else if s.GetTag() != t.GetTag() {
                // Tags must match
            } else if !t.IsFriendly() {
                // Do not acknowledge the request
            } else if s.AddFriend(tId) {
                r := clients.NewMail(0, sId, m.GetCode(), nil)
                admin.SendTo(sId, r)
            }
        case REMOVE_FRIEND:
            tId := m.GetTarget()
            if s.RemoveFriend(tId) {
                r := clients.NewMail(0, sId, m.GetCode(), nil)
                admin.SendTo(sId, r)
            }
        }
    }
}

