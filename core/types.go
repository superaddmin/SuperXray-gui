// Package core defines the minimal multi-core runtime contract used by the
// SuperXray panel API.
package core

import (
	"context"
	"errors"
)

type CoreType string

const (
	CoreTypeXray    CoreType = "xray"
	CoreTypeSingBox CoreType = "sing-box"
)

type State string

const (
	StateUnknown       State = "unknown"
	StateRunning       State = "running"
	StateStopped       State = "stopped"
	StateError         State = "error"
	StateNotInstalled  State = "not-installed"
	StateNotConfigured State = "not-configured"
)

var (
	ErrInstanceAlreadyRegistered = errors.New("core instance already registered")
	ErrInstanceNotFound          = errors.New("core instance not found")
	ErrInvalidInstance           = errors.New("invalid core instance")
	ErrLifecycleUnsupported      = errors.New("core lifecycle is not supported")
)

type Capabilities struct {
	Read                    bool `json:"read"`
	Write                   bool `json:"write"`
	Validate                bool `json:"validate"`
	Start                   bool `json:"start"`
	Stop                    bool `json:"stop"`
	Restart                 bool `json:"restart"`
	LifecycleViaCoreManager bool `json:"lifecycleViaCoreManager"`
}

type Status struct {
	State     State  `json:"state"`
	Version   string `json:"version"`
	ErrorMsg  string `json:"errorMsg,omitempty"`
	PID       int    `json:"pid,omitempty"`
	Binary    string `json:"binary,omitempty"`
	Config    string `json:"config,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type Instance struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	DisplayName      string       `json:"displayName"`
	CoreType         CoreType     `json:"coreType"`
	Mode             string       `json:"mode"`
	Source           string       `json:"source"`
	LifecycleOwner   string       `json:"lifecycleOwner"`
	Status           Status       `json:"status"`
	Capabilities     Capabilities `json:"capabilities"`
	WriteSupported   bool         `json:"writeSupported"`
	ManagerAttached  bool         `json:"managerAttached"`
	ExperimentalOnly bool         `json:"experimentalOnly"`
}

type LifecycleResult struct {
	State    State  `json:"state"`
	Msg      string `json:"msg,omitempty"`
	ErrorMsg string `json:"errorMsg,omitempty"`
	PID      int    `json:"pid,omitempty"`
}

type Adapter interface {
	Instance() Instance
	Status(ctx context.Context) (Status, error)
	Validate(ctx context.Context) (LifecycleResult, error)
	Start(ctx context.Context) (LifecycleResult, error)
	Stop(ctx context.Context) (LifecycleResult, error)
	Restart(ctx context.Context) (LifecycleResult, error)
}
