package core

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type Manager struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
	order    []string
}

type CoreManager = Manager

func NewManager() *Manager {
	return &Manager{
		adapters: make(map[string]Adapter),
	}
}

func NewCoreManager() *CoreManager {
	return NewManager()
}

func (m *Manager) Register(adapter Adapter) error {
	if adapter == nil {
		return ErrInvalidInstance
	}
	instance := adapter.Instance()
	id := strings.TrimSpace(instance.ID)
	if id == "" {
		return ErrInvalidInstance
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.adapters[id]; exists {
		return ErrInstanceAlreadyRegistered
	}
	m.adapters[id] = adapter
	m.order = append(m.order, id)
	sort.Strings(m.order)
	return nil
}

func (m *Manager) List(ctx context.Context) ([]Instance, error) {
	m.mu.RLock()
	ids := append([]string(nil), m.order...)
	adapters := make(map[string]Adapter, len(m.adapters))
	for id, adapter := range m.adapters {
		adapters[id] = adapter
	}
	m.mu.RUnlock()

	instances := make([]Instance, 0, len(ids))
	for _, id := range ids {
		instance, err := instanceWithStatus(ctx, adapters[id])
		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, nil
}

func (m *Manager) Get(ctx context.Context, id string) (Instance, error) {
	adapter, err := m.adapter(id)
	if err != nil {
		return Instance{}, err
	}
	return instanceWithStatus(ctx, adapter)
}

func (m *Manager) Status(ctx context.Context, id string) (Status, error) {
	adapter, err := m.adapter(id)
	if err != nil {
		return Status{}, err
	}
	return adapter.Status(ctx)
}

func (m *Manager) Validate(ctx context.Context, id string) (LifecycleResult, error) {
	adapter, err := m.adapter(id)
	if err != nil {
		return LifecycleResult{}, err
	}
	return adapter.Validate(ctx)
}

func (m *Manager) Start(ctx context.Context, id string) (LifecycleResult, error) {
	adapter, err := m.adapter(id)
	if err != nil {
		return LifecycleResult{}, err
	}
	return adapter.Start(ctx)
}

func (m *Manager) Stop(ctx context.Context, id string) (LifecycleResult, error) {
	adapter, err := m.adapter(id)
	if err != nil {
		return LifecycleResult{}, err
	}
	return adapter.Stop(ctx)
}

func (m *Manager) Restart(ctx context.Context, id string) (LifecycleResult, error) {
	adapter, err := m.adapter(id)
	if err != nil {
		return LifecycleResult{}, err
	}
	return adapter.Restart(ctx)
}

func (m *Manager) adapter(id string) (Adapter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	adapter, exists := m.adapters[strings.TrimSpace(id)]
	if !exists {
		return nil, ErrInstanceNotFound
	}
	return adapter, nil
}

func instanceWithStatus(ctx context.Context, adapter Adapter) (Instance, error) {
	instance := adapter.Instance()
	status, err := adapter.Status(ctx)
	if err != nil {
		return Instance{}, err
	}
	instance.Status = status
	return instance, nil
}
