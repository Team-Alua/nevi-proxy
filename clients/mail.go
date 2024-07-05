package clients

import (
    "encoding/binary"
)

type Mail struct {
    source uint64
    target uint64 
    code uint32
    data []byte
}

func NewEmptyMail() *Mail {
    return &Mail{}
}

func NewMail(source uint64, target uint64, code uint32, data []byte) *Mail {
    m := &Mail{}
    m.source = source
    m.target = target
    m.code = code
    m.data = data
    return m
}


func NewMailFromBytes(data []byte) *Mail {
    m := &Mail{}

    if len(data) >= 8 {
        m.source = binary.LittleEndian.Uint64(data[0:8])
    }

    if len(data) >= 16 {
        m.target = binary.LittleEndian.Uint64(data[8:16])
    }
    
    if len(data) >= 20 {
        m.code = binary.LittleEndian.Uint32(data[16:20])
        m.data = data[20:]
    }
    return m
}

func (m *Mail) ToBytes() []byte {
    data := make([]byte, 20)
    binary.LittleEndian.PutUint64(data, m.source)
    binary.LittleEndian.PutUint64(data[8:], m.target)
    binary.LittleEndian.PutUint32(data[16:], m.code)
    if m.data != nil {
        data = append(data, m.data...)
    }
    return data
}

func (m *Mail) GetSource() uint64 {
    return m.source
}

func (m *Mail) GetTarget() uint64 {
    return m.target
}

func (m *Mail) GetCode() uint32 {
    return m.code
}

func (m *Mail) GetData() []byte {
    return m.data
}


