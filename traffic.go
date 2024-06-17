package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"redis-migration-poc/redis/pool"

	"github.com/pkg/errors"
)

const (
	hash      = "tst-hash"
	keyPrefix = "key-prefix"
)

type Traffic struct {
	oldPool *pool.Pool
	newPool *pool.Pool

	currentPool *pool.Pool

	mtx sync.Mutex
}

func NewTraffic(oldPool, newPool *pool.Pool) *Traffic {
	return &Traffic{
		oldPool:     oldPool,
		newPool:     newPool,
		currentPool: oldPool,
	}
}

func (t *Traffic) Switch() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.currentPool == t.oldPool {
		t.currentPool = t.newPool
		return
	}

	t.currentPool = t.oldPool
}

func (t *Traffic) createHash(ctx context.Context, hash, key string, expire int) error {
	err := t.currentPool.CmdCtx(ctx, nil, "HINCRBY", hash, key, "1")
	if err != nil {
		return errors.Wrapf(err, "cannot increment %s-%s", hash, key)
	}

	if expire > 0 {
		err = t.currentPool.CmdCtx(ctx, nil, "EXPIRE", key, strconv.Itoa(expire))
		if err != nil {
			return errors.Wrap(err, "can't set ttl")
		}
	}
	return nil
}

func (t *Traffic) Run(ctx context.Context) {
	withoutTTL := make([]string, 0)
	for i := 0; i < 50; i++ {
		log.Printf("record #%d. written\n", i)
		key := fmt.Sprintf("%s-%d", keyPrefix, i)
		if rand.Int63()%2 == 0 {
			withoutTTL = append(withoutTTL, key)
			if err := t.createHash(ctx, hash, key, 0); err != nil {
				log.Println(err)
			}
			continue
		}

		if err := t.createHash(ctx, hash, key, 20); err != nil {
			log.Println(err)
		}
		time.Sleep(time.Second)
	}

	log.Println("keys without TTL", withoutTTL)
}
