package tcp

import (
	"context"
	"net"
)

type Handler interface {
	Handler(ctx context.Context, conn net.Conn)
	Close() error
}
