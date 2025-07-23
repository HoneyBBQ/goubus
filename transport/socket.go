package transport

import (
	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

type SocketClient struct {
}

var _ types.Transport = (*SocketClient)(nil)

// NewSocketClient creates a new socket client.
//
// Deprecated: Not implemented.
// Use NewRpcClient instead.
func NewSocketClient() *SocketClient {
	return &SocketClient{}
}

func (c *SocketClient) Call(service, method string, data any) (types.Result, error) {
	return nil, errdefs.Wrapf(errdefs.ErrNotSupported, "socket client not implemented")
}
