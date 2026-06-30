package slots

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// ErrSlotOccupied is returned by Acquire when the slot is already taken.
var ErrSlotOccupied = errors.New("slot occupied")

type slotInfo struct {
	TaskID    string    `json:"task_id"`
	StartedAt time.Time `json:"started_at"`
}

// Store is a NATS KV-backed slot registry. Each slot key is a skill name.
// TTL on the bucket acts as automatic crash recovery: if a container dies
// without releasing its slot, the key expires and the slot is freed.
type Store struct {
	kv jetstream.KeyValue
}

func New(js jetstream.JetStream, ttl time.Duration) (*Store, error) {
	kv, err := js.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket: "kha-slots",
		TTL:    ttl,
	})
	if err != nil {
		return nil, fmt.Errorf("init kv bucket: %w", err)
	}
	return &Store{kv: kv}, nil
}

// IsOccupied returns true if the skill slot has an active lock.
func (s *Store) IsOccupied(ctx context.Context, skill string) (bool, error) {
	_, err := s.kv.Get(ctx, skill)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Acquire atomically claims the slot for a task. Returns ErrSlotOccupied on race.
func (s *Store) Acquire(ctx context.Context, skill, taskID string) error {
	data, _ := json.Marshal(slotInfo{
		TaskID:    taskID,
		StartedAt: time.Now().UTC(),
	})
	_, err := s.kv.Create(ctx, skill, data)
	if errors.Is(err, jetstream.ErrKeyExists) {
		return ErrSlotOccupied
	}
	return err
}

// Release removes the slot lock so the next task can be dispatched.
func (s *Store) Release(ctx context.Context, skill string) error {
	return s.kv.Purge(ctx, skill)
}
