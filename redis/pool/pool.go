package pool

import (
	"context"
	"fmt"

	"github.com/mediocregopher/radix/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const defaultTracerName = "gitlab.sessionm.com/src/github.com/mediocregopher/radix/v4/pool"

// Pool wraps a radix pool and adds a nasty tracker to it so that we can keep contextual relations.
// If we were able to go to go-redis then tracing would be baked in.
type Pool struct {
	radix.Client
	tracer trace.Tracer
	addr   string
}

type DialFunc func(ctx context.Context, network, addr string) (radix.Conn, error)

// NewCustom is like New except you can specify a DialFunc which will be
// used when creating new connections for the pool. The common use-case is to do
// authentication for new connections.
func NewCustom(ctx context.Context, network, addr string, size int, dialer radix.Dialer) (*Pool, error) {
	poolConfig := radix.PoolConfig{
		Dialer: dialer,
		Size:   size,
	}

	client, err := poolConfig.New(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	wrapPool := Pool{Client: client}
	wrapPool.tracer = otel.Tracer(defaultTracerName)
	return &wrapPool, err
}

// Cmd automatically gets one client from the pool, executes the given command
// (returning its result), and puts the client back in the pool
func (p *Pool) Cmd(resp interface{}, cmd string, args ...string) error {
	return p.Do(context.Background(), radix.Cmd(resp, cmd, args...))
}

// CmdCtx is an interim wrapper to pass ctx to tracing.
// Calls Cmd which automatically gets one client from the pool, executes the given command
// (returning its result), and puts the client back in the pool
func (p *Pool) CmdCtx(ctx context.Context, resp interface{}, cmd string, args ...string) error {
	_, span := p.tracer.Start(ctx, cmd, trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.instance", p.addr),
			attribute.String("db.name", p.addr),
			attribute.String("db.system", "redis"),
			attribute.Int("db.redis.num_cmd", len(args)),
			attribute.String("peer.service", p.addr),
			attribute.String("db.statement", fmt.Sprintf("%v", args))))

	defer span.End()
	return p.Do(ctx, radix.Cmd(resp, cmd, args...))
}
