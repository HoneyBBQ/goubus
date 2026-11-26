//go:build unix

package goubus

import (
	"encoding/binary"
	"io"
	"math"
	"net"
	"os"
	"syscall"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

const (
	noFD               = -1
	unixFrameLeadBytes = blobmsg.HeaderBytes + blobmsg.BlobHeaderBytes
)

func supportsFDPassing(conn net.Conn) bool {
	_, ok := conn.(*net.UnixConn)

	return ok
}

func validateSendFile(sendFile *os.File) error {
	if sendFile == nil {
		return nil
	}

	rawConn, err := sendFile.SyscallConn()
	if err != nil {
		return errs.Wrapf(ErrInvalidParameter, "invalid file descriptor: %v", err)
	}

	var fdErr error
	ctrlErr := rawConn.Control(func(fd uintptr) {
		if fd > math.MaxInt {
			fdErr = errs.Wrapf(ErrInvalidParameter, "file descriptor %d out of range", fd)
		}
	})
	if ctrlErr != nil {
		return errs.Wrapf(ErrInvalidParameter, "invalid file descriptor: %v", ctrlErr)
	}

	return fdErr
}

func readWireMessage(conn net.Conn) (*blobmsg.UbusMessageHeader, []byte, int, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		hdr, payload, err := blobmsg.ReadMessage(conn)

		return hdr, payload, noFD, err
	}

	lead, recvFD, err := readUnixLead(unixConn)
	if err != nil {
		return nil, nil, noFD, err
	}

	hdr := &blobmsg.UbusMessageHeader{
		Version: lead[0],
		Type:    lead[1],
		Seq:     binary.BigEndian.Uint16(lead[2:4]),
		Peer:    binary.BigEndian.Uint32(lead[4:8]),
	}
	blobLen := binary.BigEndian.Uint32(lead[blobmsg.HeaderBytes:unixFrameLeadBytes])
	if blobLen < blobmsg.BlobHeaderBytes || blobLen > blobmsg.MaxMessageBytes {
		closeFD(recvFD)

		return nil, nil, noFD, errs.Wrapf(errs.ErrInvalidBlobLength, "message length %d", blobLen)
	}

	payload := make([]byte, blobLen)
	copy(payload[:blobmsg.BlobHeaderBytes], lead[blobmsg.HeaderBytes:unixFrameLeadBytes])
	if blobLen > blobmsg.BlobHeaderBytes {
		_, err = io.ReadFull(unixConn, payload[blobmsg.BlobHeaderBytes:])
		if err != nil {
			closeFD(recvFD)

			return nil, nil, noFD, errs.Wrapf(ErrConnectionFailed, "read blob body: %v", err)
		}
	}

	return hdr, payload, recvFD, nil
}

func readUnixLead(conn *net.UnixConn) ([unixFrameLeadBytes]byte, int, error) {
	var lead [unixFrameLeadBytes]byte
	oob := make([]byte, syscall.CmsgSpace(blobmsg.Uint32Size))
	recvFD := noFD
	offset := 0

	for offset < unixFrameLeadBytes {
		n, oobn, _, _, err := conn.ReadMsgUnix(lead[offset:], oob)
		if oobn > 0 {
			recvFD = captureFirstFD(recvFD, oob[:oobn])
		}
		offset += n
		if err != nil {
			closeFD(recvFD)

			return lead, noFD, errs.Wrapf(ErrConnectionFailed, "read header: %v", err)
		}
		if n == 0 {
			closeFD(recvFD)

			return lead, noFD, errs.Wrapf(ErrConnectionFailed, "read header: %v", io.EOF)
		}
	}

	return lead, recvFD, nil
}

func captureFirstFD(existing int, oob []byte) int {
	msgs, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return existing
	}

	for _, msg := range msgs {
		fds, parseErr := syscall.ParseUnixRights(&msg)
		if parseErr != nil {
			continue
		}
		for _, fd := range fds {
			syscall.CloseOnExec(fd)
			if existing < 0 {
				existing = fd
			} else {
				closeFD(fd)
			}
		}
	}

	return existing
}

func writeWireBytes(conn net.Conn, frame []byte, sendFile *os.File) error {
	if sendFile == nil {
		_, err := conn.Write(frame)

		return err
	}

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return ErrNotSupported
	}

	rawConn, err := sendFile.SyscallConn()
	if err != nil {
		return errs.Wrapf(ErrInvalidParameter, "inspect file descriptor: %v", err)
	}

	var writeErr error
	ctrlErr := rawConn.Control(func(fd uintptr) {
		if fd > math.MaxInt {
			writeErr = errs.Wrapf(ErrInvalidParameter, "file descriptor %d out of range", fd)

			return
		}
		writeErr = writeUnixWithFD(unixConn, frame, int(fd))
	})
	if ctrlErr != nil {
		return errs.Wrapf(ErrInvalidParameter, "inspect file descriptor: %v", ctrlErr)
	}

	return writeErr
}

func writeUnixWithFD(conn *net.UnixConn, frame []byte, fd int) error {
	oob := syscall.UnixRights(fd)
	written := 0

	for written < len(frame) {
		n, _, err := conn.WriteMsgUnix(frame[written:], oob, nil)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		written += n
		oob = nil
	}

	return nil
}

func closeFD(fd int) {
	if fd >= 0 {
		_ = syscall.Close(fd)
	}
}

func fdToFile(fd int) *os.File {
	if fd < 0 {
		return nil
	}
	_ = syscall.SetNonblock(fd, true)

	return os.NewFile(uintptr(fd), "ubus-fd")
}
