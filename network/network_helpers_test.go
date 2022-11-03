package network

import (
	"encoding/binary"
)

type WriteBuffer struct {
	Bytes []byte

	msgStart int
}

func writeStartupMsg(buf *WriteBuffer, user, database, appName string) {
	// Write startup message header
	buf.msgStart = len(buf.Bytes)
	buf.Bytes = append(buf.Bytes, 0, 0, 0, 0)

	// Write protocol version
	buf.Bytes = append(buf.Bytes, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(buf.Bytes[len(buf.Bytes)-4:], uint32(196608))

	buf.WriteString("user")
	buf.WriteString(user)
	buf.WriteString("database")
	buf.WriteString(database)
	buf.WriteString("application_name")
	buf.WriteString(appName)
	buf.WriteString("")

	// Write message length
	binary.BigEndian.PutUint32(
		buf.Bytes[buf.msgStart:], uint32(len(buf.Bytes)-buf.msgStart))
}

func (buf *WriteBuffer) WriteString(s string) {
	buf.Bytes = append(buf.Bytes, s...)
	buf.Bytes = append(buf.Bytes, 0)
}

func CreatePostgreSQLPacket(typ byte, msg []byte) []byte {
	packet := make([]byte, 1+4+len(msg))

	packet[0] = typ
	binary.BigEndian.PutUint32(packet[1:], uint32(len(msg)+4))
	for i, b := range msg {
		packet[i+5] = b
	}

	return packet
}

func CreatePgStartupPacket() []byte {
	buf := &WriteBuffer{}
	writeStartupMsg(buf, "postgres", "postgres", "gatewayd")
	return buf.Bytes
}
