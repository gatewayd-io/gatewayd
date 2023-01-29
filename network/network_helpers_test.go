package network

import "encoding/binary"

func CreatePostgreSQLPacket(typ byte, msg []byte) []byte {
	packet := make([]byte, 1+4+len(msg))

	packet[0] = typ
	binary.BigEndian.PutUint32(packet[1:], uint32(len(msg)+4))
	for i, b := range msg {
		packet[i+5] = b
	}

	return packet
}
