package main

import (
	"context"
	"log"
	"net"
	"time"

	"redis-migration-poc/redis/pool"

	"github.com/mediocregopher/radix/v4"
	"github.com/pkg/errors"
)

func redisOldPool(ctx context.Context, protocol, hostname string, maxConns int) (*pool.Pool, error) {
	pool, err := pool.NewCustom(ctx, protocol, hostname, maxConns, radix.Dialer{NetDialer: new(net.Dialer)})
	if err != nil {
		return nil, errors.Wrap(err, "can't create redis pool")
	}

	go func() {
		for {
			err := pool.CmdCtx(ctx, nil, "PING")
			if err != nil {
				log.Printf("Error pinging redis: %s", err)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	return pool, nil
}

func redisNewPool(ctx context.Context, protocol, hostname, password string, maxConns int) (*pool.Pool, error) {
	pool, err := pool.NewCustom(ctx, protocol, hostname, maxConns, radix.Dialer{
		NetDialer: new(pool.TLSDialer),
		AuthPass:  password,
	})

	if err != nil {
		return nil, errors.Wrap(err, "can't create redis pool")
	}

	go func() {
		for {
			err := pool.CmdCtx(ctx, nil, "PING")
			if err != nil {
				log.Printf("Error pinging redis: %s", err)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	return pool, nil
}

// create traffice on redis-old
// run redis-shaker to migrate data to redis-new
// switch on the fly to redis-new (in production using consul event and consul event listener)
func main() {
	log.Println("redis-migration-poc")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	oldPool, err := redisOldPool(ctx, "tcp", "0.0.0.0:6379", 5)
	if err != nil {
		log.Fatalf("cannot create connection to old redis: %v", err)
	}

	newPool, err := redisNewPool(ctx, "tcp", "0.0.0.0:7380", "redis-poc-pwd", 5)
	if err != nil {
		log.Fatalf("cannot create connection to new redis: %v", err)
	}

	traffic := NewTraffic(oldPool, newPool)
	traffic.Run(ctx)
}
