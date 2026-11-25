package blobmsg

import (
	"encoding/binary"
	"io"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

func EncodeHeader(w io.Writer, header *UbusMessageHeader) error {
	if header == nil {
		return ErrNilValue
	}

	var raw [HeaderBytes]byte
	raw[0] = header.Version
	raw[1] = header.Type
	binary.BigEndian.PutUint16(raw[2:4], header.Seq)
	binary.BigEndian.PutUint32(raw[4:8], header.Peer)

	_, err := w.Write(raw[:])

	return err
}

func ReadMessage(reader io.Reader) (*UbusMessageHeader, []byte, error) {
	var headerBytesBuf [HeaderBytes]byte

	_, err := io.ReadFull(reader, headerBytesBuf[:])
	if err != nil {
		return nil, nil, errs.Wrapf(errs.ErrConnectionFailed, "read header: %v", err)
	}

	hdr := &UbusMessageHeader{
		Version: headerBytesBuf[0],
		Type:    headerBytesBuf[1],
		Seq:     binary.BigEndian.Uint16(headerBytesBuf[2:4]),
		Peer:    binary.BigEndian.Uint32(headerBytesBuf[4:8]),
	}

	var blobHeader [BlobHeaderBytes]byte
	_, err = io.ReadFull(reader, blobHeader[:])
	if err != nil {
		return nil, nil, errs.Wrapf(errs.ErrConnectionFailed, "read blob header: %v", err)
	}

	blobLen := binary.BigEndian.Uint32(blobHeader[:])
	if blobLen < BlobHeaderBytes || blobLen > MaxMessageBytes {
		return nil, nil, errs.Wrapf(errs.ErrInvalidBlobLength, "message length %d", blobLen)
	}

	payload := make([]byte, blobLen)
	copy(payload[:BlobHeaderBytes], blobHeader[:])

	if blobLen > BlobHeaderBytes {
		_, err = io.ReadFull(reader, payload[BlobHeaderBytes:])
		if err != nil {
			return nil, nil, errs.Wrapf(errs.ErrConnectionFailed, "read blob body: %v", err)
		}
	}

	return hdr, payload, nil
}
