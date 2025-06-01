package websocket

import (
	"encoding/binary"
	"errors"
)

type Frame struct {
	FIN           bool    // 1 bit
	RSV1          bool    // 1 bit
	RSV2          bool    // 1 bit
	RSV3          bool    // 1 bit
	OpCode        Opcode  // 4 bits
	IsMasked      bool    // 1 bit
	MaskingKey    [4]byte // 0 - 4 bytes
	PayloadLength uint64  // 7 bits
	Payload       []byte

	// there are 2 more frame data which is extension and application data, but we don't need them for now
}

func DecodeFrame(data []byte) (Frame, error) {
	frame := Frame{}
	offset := 0

	// TODO: check if data is to short
	if len(data) < offset+1 {
		return frame, errors.New("data is too short")
	}

	// get the fin, rsv1, rsv2, rsv3 and opcode bits
	// since the bit size is 8, and the FIN + RSV1 + RSV2 + RSV3 + opcode is 8 bits, we can just shift the data from first byte
	// okay, where is the 0x80, 0x40, 0x20 came ?? that came from the spec.
	// so for example for FIN. because FIN located at the first bit of byte, so to get the FIN bit, we need to shift the data from first byte. that is 0x80 (hexadecimal) came.
	// after that, we use `and` operator to get the FIN.
	// here is the ilustration
	// 0x80 = 10000000
	// 0x40 = 01000000
	// 0x20 = 00100000
	// 0x10 = 00010000
	// 0x0F = 00001111
	frame.FIN = data[offset]&0x80 != 0
	frame.RSV1 = data[offset]&0x40 != 0
	frame.RSV2 = data[offset]&0x20 != 0
	frame.RSV3 = data[offset]&0x10 != 0
	frame.OpCode = Opcode(data[offset] & 0x0F)
	offset++

	// the RSV must be 0 unless the extension is negotiated
	// since our server don't support extension, so we can ignore it
	if frame.RSV1 || frame.RSV2 || frame.RSV3 {
		return frame, errors.New("RSV must be 0")
	}

	// check if the data is short
	if len(data) < offset+1 {
		return frame, errors.New("data is too short")
	}

	// get the masked and payload length
	frame.IsMasked = data[offset]&0x80 != 0   // 0x10000000
	payloadLength := int(data[offset] & 0x7F) // 0x01111111
	offset++

	switch payloadLength {
	case 126:
		if len(data) < offset+2 { // offset = 2 => 2 + 2 = 4
			return frame, errors.New("data is too short")
		}

		frame.PayloadLength = uint64(binary.BigEndian.Uint16(data[offset : offset+2])) // offset = 2 => 2 + 2 = 2 : 4
		offset += 2
	case 127:
		if len(data) < offset+8 { // offset = 2 => 2 + 8 = 10
			return frame, errors.New("data is too short")
		}

		frame.PayloadLength = uint64(binary.BigEndian.Uint64(data[offset : offset+8])) // offset = 2 => 2 + 8 = 10
		offset += 8

	default:
		frame.PayloadLength = uint64(payloadLength)
	}

	// TODO: check if the data is too long

	// read the masked key if mask bit is 1
	if frame.IsMasked {
		if len(data) < offset+4 { // offset = either 2, 4, or 10 => 2 + 4 = 6
			return frame, errors.New("data is too short")
		}

		// use copy because we set fixed sized on the struct
		copy(frame.MaskingKey[:], data[offset:offset+4])
		offset += 4
	}

	// read payload
	if len(data) < offset+int(frame.PayloadLength) {
		return frame, errors.New("data is too short")
	}

	payloadStart := offset                          // offset = either 6, 10, or 14
	payloadEnd := offset + int(frame.PayloadLength) // offset = either 6, 10, or 14 => 6 + payload length
	frame.Payload = data[payloadStart:payloadEnd]   // read from eighter 6, 10, 14 until payloadEnd
	offset = payloadEnd

	if frame.IsMasked {
		for i := range frame.Payload {
			// xor with masking key, see https://datatracker.ietf.org/doc/html/rfc6455#section-5.3
			// j                   = i MOD 4
			// transformed-octet-i = original-octet-i XOR masking-key-octet-j
			// j is mean masking key, so we need to use i%4 to get the right key
			// then we can use xor operator to get the transformed octet
			frame.Payload[i] = frame.Payload[i] ^ frame.MaskingKey[i%4]
		}
	}

	return frame, nil
}

func EncodeFrame(payload []byte, opcode Opcode) ([]byte, error) {
	// for the size, since we know the msg frame size, we can just use 14 as the initial size + payload length
	// 1 byte (FIN/Opcode)
	// + 1 byte (Mask bit/Initial length)
	// + 8 bytes (for 64-bit extended length)
	// + 4 bytes (for masking key)
	// = 14 bytes
	msgFrame := make([]byte, 0, 14+len(payload))

	// set the FIN, RSV1, RSV2, RSV3,
	// example: 0x80 | 0x1 => 0x81 this is 1 bytes (FIN, RSV1, RSV2, RSV3 is 1 bit for each and opcode is 4 bits)
	// so the binary result look like this:
	// 10000000  (0x80)
	// | 00000001  (0x1)
	// ------------------
	// 10000001  (0x81)
	// since we used the encodeFrame only for server, so the masked is always false
	msgFrame = append(msgFrame, (byte(0x80) | byte(opcode)))

	payloadLen := len(payload)
	var extendedPayloadLen []byte // this used to hold the extended payload length

	if payloadLen <= 125 {
		msgFrame = append(msgFrame, byte(payloadLen))
	} else if payloadLen <= 65535 { // 65535 is the max value for uint16 (0xFFFF)
		msgFrame = append(msgFrame, 126)

		extendedPayloadLen = make([]byte, 2)
		binary.BigEndian.PutUint16(extendedPayloadLen, uint16(payloadLen))
	} else {
		msgFrame = append(msgFrame, 127)

		extendedPayloadLen = make([]byte, 8)
		binary.BigEndian.PutUint64(extendedPayloadLen, uint64(payloadLen))
	}

	if extendedPayloadLen != nil {
		msgFrame = append(msgFrame, extendedPayloadLen...)
	}

	msgFrame = append(msgFrame, payload...)

	return msgFrame, nil
}
