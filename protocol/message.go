package protocol

import (
    "fmt"
)

func (p *Protocol) HandleMail() {
    // clientList := p.clientList
    mailer := p.mailer
    for {
        msg := <-mailer
        m := NewMailFromBytes(msg)
        fmt.Println(m.source, m.target, m.code, m.data)
    }
}
