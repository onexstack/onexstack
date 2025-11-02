package flux

import (
	"fmt"
	"time"
)

// Option represents a configuration option for Flux cache
type Option[K Key, V any] func(*Config[K, V]) error

// WithMaxKeys sets the maximum number of keys in cache
func WithMaxKeys[K Key, V any](maxKeys int64) Option[K, V] {
	return func(config *Config[K, V]) error {
		if maxKeys <= 0 {
			return fmt.Errorf("maxKeys must be positive, got %d", maxKeys)
		}
		config.MaxKeys = maxKeys
		return nil
	}
}

// WithTTL sets the time-to-live for cache entries
func WithTTL[K Key, V any](ttl time.Duration) Option[K, V] {
	return func(config *Config[K, V]) error {
		if ttl < 0 {
			return fmt.Errorf("ttl cannot be negative, got %v", ttl)
		}
		config.TTL = ttl
		return nil
	}
}

// WithLoader sets the data loader interface
func WithLoader[K Key, V any](loader Loader[K, V]) Option[K, V] {
	return func(config *Config[K, V]) error {
		if loader == nil {
			return fmt.Errorf("loader cannot be nil")
		}
		config.Loader = loader
		return nil
	}
}

// WithLoadTimeout sets the timeout for load operations
func WithLoadTimeout[K Key, V any](timeout time.Duration) Option[K, V] {
	return func(config *Config[K, V]) error {
		if timeout <= 0 {
			return fmt.Errorf("load timeout must be positive, got %v", timeout)
		}
		config.LoadTimeout = timeout
		return nil
	}
}

// WithAsyncRefresh enables async refresh with specified interval
func WithAsyncRefresh[K Key, V any](interval time.Duration) Option[K, V] {
	return func(config *Config[K, V]) error {
		if interval <= 0 {
			return fmt.Errorf("refresh interval must be positive, got %v", interval)
		}
		config.EnableAsync = true
		config.RefreshInterval = interval
		return nil
	}
}

// WithEnableAsync enables/disable async refresh with specified interval.
func WithEnableAsync[K Key, V any](enabled bool) Option[K, V] {
	return func(config *Config[K, V]) error {
		config.EnableAsync = enabled
		return nil
	}
}

// WithRefreshMode sets the refresh mode (single or batch)
func WithRefreshMode[K Key, V any](mode RefreshMode) Option[K, V] {
	return func(config *Config[K, V]) error {
		config.RefreshMode = mode
		return nil
	}
}

// WithMaxConcurrency sets the maximum concurrent refresh operations
func WithMaxConcurrency[K Key, V any](maxConcurrency int) Option[K, V] {
	return func(config *Config[K, V]) error {
		if maxConcurrency <= 0 {
			return fmt.Errorf("maxConcurrency must be positive, got %d", maxConcurrency)
		}
		config.MaxConcurrency = maxConcurrency
		return nil
	}
}

// WithBatchSize sets the batch size for batch refresh mode
func WithBatchSize[K Key, V any](batchSize int) Option[K, V] {
	return func(config *Config[K, V]) error {
		if batchSize <= 0 {
			return fmt.Errorf("batchSize must be positive, got %d", batchSize)
		}
		config.BatchSize = batchSize
		return nil
	}
}

// WithBufferItems sets the buffer size for ristretto cache
func WithBufferItems[K Key, V any](bufferItems int64) Option[K, V] {
	return func(config *Config[K, V]) error {
		if bufferItems <= 0 {
			return fmt.Errorf("bufferItems must be positive, got %d", bufferItems)
		}
		config.BufferItems = bufferItems
		return nil
	}
}

// WithStats enables or disables statistics collection
func WithStats[K Key, V any](enable bool) Option[K, V] {
	return func(config *Config[K, V]) error {
		config.EnableStats = enable
		return nil
	}
}

// WithAsyncConfig sets all async-related configurations at once
func WithAsyncConfig[K Key, V any](enable bool, interval time.Duration, mode RefreshMode, maxConcurrency int, batchSize int) Option[K, V] {
	return func(config *Config[K, V]) error {
		if enable {
			if interval <= 0 {
				return fmt.Errorf("refresh interval must be positive when async is enabled, got %v", interval)
			}
			if maxConcurrency <= 0 {
				return fmt.Errorf("maxConcurrency must be positive when async is enabled, got %d", maxConcurrency)
			}
			if batchSize <= 0 {
				return fmt.Errorf("batchSize must be positive, got %d", batchSize)
			}
		}

		config.EnableAsync = enable
		config.RefreshInterval = interval
		config.RefreshMode = mode
		config.MaxConcurrency = maxConcurrency
		config.BatchSize = batchSize
		return nil
	}
}

// WithConfig sets the entire configuration (useful for custom configs)
func WithConfig[K Key, V any](cfg *Config[K, V]) Option[K, V] {
	return func(config *Config[K, V]) error {
		if cfg == nil {
			return fmt.Errorf("config cannot be nil")
		}
		*config = *cfg
		return nil
	}
}
