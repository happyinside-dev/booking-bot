package fsm

import "testing"

func TestRedisKeysAreIsolatedByBot(t *testing.T) {
	t.Parallel()
	first := &RedisStore{scope: "bot-1"}
	second := &RedisStore{scope: "bot-2"}
	if first.key(42) == second.key(42) {
		t.Fatal("same Telegram user collided across bots")
	}
	if first.key(42) == first.key(43) {
		t.Fatal("different Telegram users collided")
	}
}
