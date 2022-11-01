package network

import "encoding/binary"

func CreatePostgreSQLPacket(typ byte, msg []byte) []byte {
	var packet []byte

	packet = append(packet, typ)
	binary.BigEndian.PutUint32(packet[1:], uint32(len(msg)+4))
	packet = append(packet, msg...)

	return packet
}
