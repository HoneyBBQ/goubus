//go:build !unix

package goubus

import (
	"net"
	"os"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

const noFD = -1

func supportsFDPassing(net.Conn) bool {
	return false
}

func validateSendFile(*os.File) error {
	return nil
}

func readWireMessage(conn net.Conn) (*blobmsg.UbusMessageHeader, []byte, int, error) {
	hdr, payload, err := blobmsg.ReadMessage(conn)

	return hdr, payload, noFD, err
}

func writeWireBytes(conn net.Conn, frame []byte, sendFile *os.File) error {
	if sendFile != nil {
		return ErrNotSupported
	}
	_, err := conn.Write(frame)

	return err
}

func closeFD(int) {}

func fdToFile(int) *os.File {
	return nil
}
