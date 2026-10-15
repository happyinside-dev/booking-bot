package fsm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("fsm state not found")

type State struct {
	Flow string            `json:"flow"`
	Step string            `json:"step"`
	Data map[string]string `json:"data,omitempty"`
}

type Store interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, state *State) error
	Delete(ctx context.Context, telegramID int64) error
}

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
	scope  string
}

func NewRedisStore(client *redis.Client, ttl time.Duration, scope string) *RedisStore {
	return &RedisStore{client: client, ttl: ttl, scope: scope}
}

func (s *RedisStore) key(id int64) string { return fmt.Sprintf("booking-bot:fsm:%s:%d", s.scope, id) }

func (s *RedisStore) Get(ctx context.Context, telegramID int64) (*State, error) {
	raw, err := s.client.Get(ctx, s.key(telegramID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fsm: get state: %w", err)
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("fsm: decode state: %w", err)
	}
	if state.Data == nil {
		state.Data = make(map[string]string)
	}
	return &state, nil
}

func (s *RedisStore) Set(ctx context.Context, telegramID int64, state *State) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("fsm: encode state: %w", err)
	}
	if err := s.client.Set(ctx, s.key(telegramID), raw, s.ttl).Err(); err != nil {
		return fmt.Errorf("fsm: set state: %w", err)
	}
	return nil
}

func (s *RedisStore) Delete(ctx context.Context, telegramID int64) error {
	if err := s.client.Del(ctx, s.key(telegramID)).Err(); err != nil {
		return fmt.Errorf("fsm: delete state: %w", err)
	}
	return nil
}
