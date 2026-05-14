// Package core provides the fundamental building blocks of v2ray.
// It defines the core interfaces and structures used throughout the project.
package core

import (
	"context"
	"sync"

	"google.golang.org/protobuf/proto"
)

// Version information for the v2ray core.
const (
	VersionMajor = 5
	VersionMinor = 0
	VersionPatch = 0
	VersionBuild = 1
)

// Version returns the current version of v2ray core as a string.
func Version() string {
	return "5.0.0"
}

// Instance is the core instance of v2ray.
// It manages the lifecycle of all features and services.
type Instance struct {
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	features []Feature
	running  bool
}

// Feature is the interface implemented by all v2ray features.
type Feature interface {
	// Type returns the type of the feature.
	Type() interface{}
	// Start initializes and starts the feature.
	Start() error
	// Close shuts down the feature.
	Close() error
}

// Config is the top-level configuration for a v2ray instance.
type Config struct {
	// Raw protobuf message for the configuration.
	raw proto.Message
}

// New creates a new v2ray instance with the given configuration.
func New(config *Config) (*Instance, error) {
	ctx, cancel := context.WithCancel(context.Background())
	inst := &Instance{
		ctx:    ctx,
		cancel: cancel,
	}
	return inst, nil
}

// NewWithContext creates a new v2ray instance using the provided context.
func NewWithContext(ctx context.Context, config *Config) (*Instance, error) {
	ctx, cancel := context.WithCancel(ctx)
	inst := &Instance{
		ctx:    ctx,
		cancel: cancel,
	}
	return inst, nil
}

// AddFeature registers a feature to this instance.
// Features must be added before calling Start.
func (i *Instance) AddFeature(feature Feature) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.running {
		return newError("cannot add feature to a running instance")
	}
	i.features = append(i.features, feature)
	return nil
}

// Start initializes and starts all registered features.
func (i *Instance) Start() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.running {
		return newError("instance is already running")
	}

	for _, f := range i.features {
		if err := f.Start(); err != nil {
			return newError("failed to start feature").Base(err)
		}
	}

	i.running = true
	return nil
}

// Close shuts down the instance and all registered features.
func (i *Instance) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.running {
		return newError("instance is not running")
	}

	i.cancel()

	var errs []error
	for idx := len(i.features) - 1; idx >= 0; idx-- {
		if err := i.features[idx].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	i.running = false

	if len(errs) > 0 {
		return newError("errors occurred while closing instance").Base(errs[0])
	}
	return nil
}

// Context returns the context associated with this instance.
func (i *Instance) Context() context.Context {
	return i.ctx
}

// IsRunning returns true if the instance is currently running.
func (i *Instance) IsRunning() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.running
}
