package pool

import (
	"context"
	"crypto/tls"
	"net"
	"time"
)

type TLSDialer func(context.Context, string, string) (net.Conn, error)

func (td TLSDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, network, addr, &tls.Config{
		InsecureSkipVerify: true,
	})
}
