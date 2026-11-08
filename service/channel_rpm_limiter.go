package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/go-redis/redis/v8"
)

const channelRPMWindowSeconds = 60

var (
	ErrChannelRPMRedisUnavailable = errors.New("channel rpm limiter redis unavailable")
	ErrChannelRPMLimitReached     = errors.New("channel rpm limit reached")
)

// channelRPMWindowScript increments a channel/key counter and initializes its
// fixed 60-second window atomically. Keeping the increment and TTL in one Lua
// script prevents concurrent New API replicas from both passing the same slot.
const channelRPMWindowScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
if count <= tonumber(ARGV[1]) then
  return 1
end
return 0
`

func channelRPMLimitKey(channelID, keyIndex int) string {
	return fmt.Sprintf("newapi:channel-rpm:%d:%d", channelID, keyIndex)
}

// AllowChannelRPMLimit reserves one upstream request slot. A nil/zero limit
// is handled by callers as the legacy unlimited path. Redis errors fail closed
// for configured channels so a topology failure cannot bypass the policy.
func AllowChannelRPMLimit(ctx context.Context, channelID, keyIndex, limit int) (bool, error) {
	if limit <= 0 {
		return true, nil
	}
	if common.RDB == nil {
		return false, ErrChannelRPMRedisUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := common.RDB.Eval(ctx, channelRPMWindowScript, []string{channelRPMLimitKey(channelID, keyIndex)}, limit, channelRPMWindowSeconds).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, ErrChannelRPMRedisUnavailable
		}
		return false, fmt.Errorf("%w: %v", ErrChannelRPMRedisUnavailable, err)
	}
	return result == 1, nil
}
