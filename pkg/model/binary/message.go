package binary

import (
	"bufio"
	"encoding/binary"
	"io"
	"math"
)

type Message struct {
	Version uint64
	Parent1 [32]byte
	Parent2 [32]byte
	Payload Payload
	Nonce   uint64
}

func ReadMessage(r io.Reader) (*Message, error) {

	reader := bufio.NewReader(r)

	version, err := readVarintInRange(r, 127)
	if err != nil {
		return nil, err
	}

	if version != 1 {
		return nil, ErrUnsupportedVersion
	}

	parent1, err := readBytes(r, 32)
	if err != nil {
		return nil, err
	}

	parent2, err := readBytes(r, 32)
	if err != nil {
		return nil, err
	}

	payload, err := readPayload(r)
	if err != nil {
		if err == ErrEmptyPayload {
			// We are expecting to have a payload here
			err = ErrInvalidPayloadLength
		}
		return nil, err
	}

	nonce, err := readUint64(reader)
	if err != nil {
		return nil, err
	}

	message := &Message{
		Version: 1,
		Payload: payload,
		Nonce:   nonce,
	}
	copy(message.Parent1[:], parent1)
	copy(message.Parent2[:], parent2)

	return message, nil
}

func readPayload(r io.Reader) (Payload, error) {

	reader := bufio.NewReader(r)

	// TODO: define max
	payloadLength, err := readVarintInRange(reader, math.MaxUint64)
	if err != nil {
		return nil, err
	}

	if payloadLength == 0 {
		return nil, ErrEmptyPayload
	}

	// Peek the payload type
	payloadTypeBytes, err := reader.Peek(10)
	payloadType, n := binary.Uvarint(payloadTypeBytes)
	if n < 0 {
		return nil, ErrWrongPayloadType
	}

	// Create a reader that reads at most the payload length,
	// so we can pass it over without the risk of it consuming the nonce
	payloadReader := io.LimitReader(reader, int64(payloadLength))

	switch PayloadType(payloadType) {

	case PayloadTypeSignedTransaction:
		return readSignedTransaction(payloadReader)

	case PayloadTypeMilestone:
		return readMilestone(payloadReader)

	case PayloadTypeUnsignedData:
		return readUnsignedData(payloadReader)

	case PayloadTypeSignedData:
		return readSignedData(payloadReader)

	case PayloadTypeIndexation:
		return readIndexation(payloadReader)

	default:
		// ignore the payload data but do not return an error, we need to keep the message around
		reader.Discard(int(payloadLength))
		unsupported := &UnsupportedPayload{
			payloadType: PayloadType(payloadType),
		}
		return unsupported, nil
	}
}