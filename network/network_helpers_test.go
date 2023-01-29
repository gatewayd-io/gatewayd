package network

import "encoding/binary"

func CreatePostgreSQLPacket(typ byte, msg []byte) []byte {
	packet := make([]byte, 1+4+len(msg))

	packet = append(packet, typ)
	binary.BigEndian.PutUint32(packet, uint32(len(msg)+4))
	packet = append(packet, msg...)

	return packet
}
