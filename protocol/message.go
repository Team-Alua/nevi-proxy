package protocol

import (
    "encoding/binary"
)


const (
    SEND_MESSAGE = 1
    UPDATE_STATE = 2
    GET_CLIENTS_WITH_TAG = 3
)

func (p *Protocol) notify(id uint64, data []byte) {
    c := p.clientList.GetClient(id)
    if c == nil {
        return
    }
    c.SendMail(data)
}

func (p *Protocol) notifyAll(ids []uint64, data []byte) {
    for _, id := range ids {
        p.notify(id, data)
    }
}

func (p *Protocol) HandleMail() {
    clientList := p.clientList
    mailer := p.mailer
    for {
        msg := <-mailer
        m := NewMailFromBytes(msg)
        s := clientList.GetClient(m.source)
        if s == nil {
            continue
        } 
        switch m.code {
        case SEND_MESSAGE:
            sId := m.source
            tId := m.target
            t := clientList.GetClient(tId)
            var bounced bool
            // must be a valid client
            if t == nil {
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
                fm := NewMail(0, tId, m.code, nil)
                p.notify(sId, fm.ToBytes())
            } else {
                // Mail forwarded to target
                fm := NewMail(sId, tId, m.code, m.data)
                p.notify(tId, fm.ToBytes())
            }
        case UPDATE_STATE:
            s.SetState(m.target)
            if s.IsServing() {
                tag := s.GetTag()
                ids := clientList.GetClientsWithTag(tag)
                r := NewMail(0, s.Id, m.code, nil)
                msg := r.ToBytes()
                for _, id := range ids {
                    n := clientList.GetClient(id)
                    // Ignore servers and ignore clients
                    // who do not want friends
                    if n.IsServing() || !n.IsFriendly() {
                        continue
                    }
                    p.notify(id, msg)
                }
            }
            r := NewMail(0, m.target, m.code, nil)
            p.notify(s.Id, r.ToBytes())
        case GET_CLIENTS_WITH_TAG: 
            ids := clientList.GetClientsWithTag(m.target)
            payload := make([]byte, len(ids) * 8)
            for idx, id := range ids {
                binary.LittleEndian.PutUint64(payload[8 * idx:], id)
            }

            r := NewMail(0, s.Id, m.code, payload)
            p.notify(s.Id, r.ToBytes())
        }
    }
}

