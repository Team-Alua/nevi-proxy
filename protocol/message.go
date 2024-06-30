package protocol

import (
    "encoding/binary"

    "github.com/Team-Alua/nevi-proxy/clients"
)


const (
    SEND_MESSAGE = 1
    UPDATE_TAG = 2
    GET_CLIENTS_WITH_TAG = 3
)

func (p *Protocol) notify(id uint64, data []byte) {
    c := p.clientList.GetClient(id)
    if c == nil {
        return
    }
    msg := clients.NewBinaryMessage(data)
    c.GetWriter().Write(msg)
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
            t := clientList.GetClient(m.target)
            if t == nil {
                // Mail bounced
                fm := NewMail(0, m.target, m.code, nil)
                p.notify(m.source, fm.ToBytes())
            } else {
                // Mail forwarded to target
                fm := NewMail(m.source, m.target, m.code, m.data)
                p.notify(m.target, fm.ToBytes())
            }
        case UPDATE_TAG:
            s.Tag = m.target
            if isTagServer(m.target) {
                tag := getListenerTag(m.target)
                ids := clientList.GetClientsWithTag(tag)
                r := NewMail(0, s.Id, m.code, nil)
                p.notifyAll(ids, r.ToBytes())
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

