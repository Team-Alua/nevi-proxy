package clients

import (
    "encoding/binary"
    "time"

    "github.com/gorilla/websocket"

    "github.com/Team-Alua/nevi-proxy/isync"
)

type Client struct {
    id uint64 // Unique identifier 
    mailer chan<-[]byte

    // State format
    // XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXYZ
    // X - 62 bit tag value
    // Y - {1 = listening 0 = not listening} for new clients (friends)
    // Z - {1 = server 0 = client} 
    tag uint64 // 62 bits - 0 tag value reserved for no tag
    friendly bool
    serving bool

    // All clients that are allowed to send a message when
    // the client is not friendly
    friends [32]uint64 

    conn *websocket.Conn
    connected *isync.SetGetter[bool]
    writer *isync.SetGetter[*isync.ReadWriter[*Message]]
}

func NewClient(conn *websocket.Conn) *Client {
    var c Client

    c.conn = conn
    c.connected = isync.NewSetGetter[bool]()
    c.connected.Set(true)
    c.writer = isync.NewSetGetter[*isync.ReadWriter[*Message]]()
    c.writer.Set(isync.NewReadWriter[*Message]())

    return &c
}

func (c *Client) SetId(id uint64) {
    if c == nil {
        return
    }
    c.id = id
}

func (c *Client) SetMailer(mailer chan<-[]byte) {
    if c == nil {
        return
    }
    c.mailer = mailer
}

func (c *Client) SetState(state uint64) {
    if c == nil {
        return
    }
    c.tag = (state & ^(uint64(0b11) << 62)) 
    c.friendly = ((state & 1 << 62) >> 62) == 1
    c.serving = ((state & 1 << 63) >> 63) == 1
}

func (c *Client) GetTag() uint64 {
    if c == nil {
        return 0
    }
    return c.tag
}

func (c *Client) IsFriendly() bool {
    if c == nil {
        return false
    }
    return c.friendly
}

func (c *Client) IsServing() bool {
    if c == nil {
        return false
    }
    return c.serving
}

func (c *Client) getFreeFriendIndex() int {
    if c == nil {
        return -1
    }

    for idx, id := range c.friends {
        if id == 0 {
            return idx
        }
    }
    return -1
}

func (c *Client) AddFriend(friendId uint64) bool {
    if c == nil {
        return false
    }
    
    // Can't add yourself as a friend
    if c.id == friendId {
        return false
    }

    if !c.FriendsWith(friendId) {
        idx := c.getFreeFriendIndex()
        if idx == -1 {
            return false
        }
        c.friends[idx] = friendId
        return true
    }
    return false
}

func (c *Client) getFriendIndex(friendId uint64) int {
    if c == nil {
        return -1
    }

    for idx, id := range c.friends {
        if id == friendId {
            return idx
        }
    }
    return -1
}

func (c *Client) RemoveFriend(friendId uint64) bool {
    if c == nil {
        return false
    }
    idx := c.getFriendIndex(friendId)
    if idx == -1 {
        return false
    }
    c.friends[idx] = 0
    return true
}


func (c *Client) FriendsWith(friendId uint64) bool {
    if c == nil {
        return false
    }
    return c.getFriendIndex(friendId) != -1
}

func (c *Client) IsConnected() bool {
    if c == nil {
        return false
    }

    return c.connected.Get()
}

func (c *Client) GetWriter() *isync.ReadWriter[*Message] {
    if c == nil {
        return nil
    }
    return c.writer.Get()
}

func (c *Client) Close() {
    if c == nil {
        return
    }

    if c.connected.Exchange(false) {
        c.writer.Get().Close()
        c.conn.Close()  
    }
}

// Listen for client messages
func (c *Client) Listen() {
    if c == nil {
        return
    }

    pongWait := 60 * time.Second
    c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
    for {
        // Reset to wait forever
        c.conn.SetReadDeadline(time.Time{})
        c.conn.SetReadLimit(1024)
        msgType, data, err := c.conn.ReadMessage()
        if err != nil {
            break
        }

        if msgType == websocket.BinaryMessage { 
            bData := make([]byte, 8)
            // Add Client Id
            binary.LittleEndian.PutUint64(bData[0:], c.id)
            bData = append(bData, data...)
            c.mailer <- bData
        } else if msgType == websocket.CloseMessage {
            break
        } else if msgType == websocket.PingMessage {
            // Respond back with a pong
            msg := NewPongMessage()
            c.writer.Get().Write(msg)
        } else {
            // Ignore for now
        }
    }
    c.Close()
}

func (c *Client) SendMail(data []byte) {
    msg := NewBinaryMessage(data)
    c.writer.Get().Write(msg)
}

// Repeat all messages sent by the server
func (c *Client) Repeat() {
    if c == nil {
        return
    }

    writeWait := 10 * time.Second

    for {
        msg := c.writer.Get().Read()
        if msg == nil {
            break
        }
        c.conn.SetWriteDeadline(time.Now().Add(writeWait))
        if err := c.conn.WriteMessage(msg.Type, msg.Data); err != nil {
            break
        }
    }
    c.Close()
}

