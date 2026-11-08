package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestAllowChannelRPMLimitUsesAtomicWindowAndKeyIsolation(t *testing.T) {
	server := miniredis.RunT(t)
	previous := common.RDB
	common.RDB = redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = common.RDB.Close()
		common.RDB = previous
	})

	ctx := context.Background()
	allowed, err := AllowChannelRPMLimit(ctx, 17, 0, 1)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = AllowChannelRPMLimit(ctx, 17, 0, 1)
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = AllowChannelRPMLimit(ctx, 17, 1, 1)
	require.NoError(t, err)
	require.True(t, allowed)

	server.FastForward(channelRPMWindowSeconds * time.Second)
	allowed, err = AllowChannelRPMLimit(ctx, 17, 0, 1)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestAllowChannelRPMLimitConcurrentSingleSlot(t *testing.T) {
	server := miniredis.RunT(t)
	previous := common.RDB
	common.RDB = redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = common.RDB.Close()
		common.RDB = previous
	})

	const callers = 16
	results := make(chan bool, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := AllowChannelRPMLimit(context.Background(), 23, 0, 1)
			if err != nil {
				results <- false
				return
			}
			results <- allowed
		}()
	}
	wg.Wait()
	close(results)

	allowedCount := 0
	for allowed := range results {
		if allowed {
			allowedCount++
		}
	}
	require.Equal(t, 1, allowedCount)
}

func TestAllowChannelRPMLimitFailsClosedWithoutRedis(t *testing.T) {
	previous := common.RDB
	common.RDB = nil
	t.Cleanup(func() { common.RDB = previous })

	allowed, err := AllowChannelRPMLimit(context.Background(), 1, 0, 1)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrChannelRPMRedisUnavailable))
	require.False(t, allowed)

	allowed, err = AllowChannelRPMLimit(context.Background(), 1, 0, 0)
	require.NoError(t, err)
	require.True(t, allowed)
}
