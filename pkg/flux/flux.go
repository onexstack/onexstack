package flux

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// Common errors
var (
	ErrCacheClosed                = errors.New("flux cache is closed")
	ErrLoaderNotSet               = errors.New("data loader is not set")
	ErrInvalidTimeout             = errors.New("invalid timeout duration")
	ErrKeyNotFound                = errors.New("key not found")
	ErrLoaderMethodNotImplemented = errors.New("required loader method not implemented")
)

// Key constraint interface that satisfies both ristretto.Key and comparable requirements
type Key interface {
	ristretto.Key
	comparable
}

// RefreshMode defines how keys are refreshed
type RefreshMode int

const (
	// RefreshSingle refreshes keys one by one
	RefreshSingle RefreshMode = iota
	// RefreshBatch refreshes keys in batches
	RefreshBatch
	// RefreshAll refreshes all keys
	RefreshAll
	// RefreshAuto automatically selects the best refresh method (LoadAll > LoadBatch > Load)
	RefreshAuto
)

// Loader defines the interface for loading data from external sources
type Loader[K Key, V any] interface {
	// Load loads a single value by key (required for RefreshSingle mode)
	Load(ctx context.Context, key K) (V, error)
}

// BatchLoader defines the interface for batch loading
type BatchLoader[K Key, V any] interface {
	Loader[K, V]
	// LoadBatch loads multiple values by keys (required for RefreshBatch mode)
	LoadBatch(ctx context.Context, keys []K) (map[K]V, error)
}

// AllLoader defines the interface for loading all data
type AllLoader[K Key, V any] interface {
	Loader[K, V]
	// LoadAll loads all available key-value pairs (highest priority in RefreshAuto mode)
	LoadAll(ctx context.Context) (map[K]V, error)
}

// FullLoader combines all loader interfaces
type FullLoader[K Key, V any] interface {
	AllLoader[K, V]
	BatchLoader[K, V]
}

// Stats represents cache statistics
type Stats struct {
	Hits          uint64 // Cache hits
	Misses        uint64 // Cache misses
	Loads         uint64 // Data loads from source
	LoadErrors    uint64 // Load errors
	Refreshes     uint64 // Async refreshes
	RefreshErrors uint64 // Refresh errors
	Keys          uint64 // Current number of keys
	Size          uint64 // Current cache size
}

// Config holds all configuration options for Flux cache
type Config[K Key, V any] struct {
	// Cache settings
	MaxKeys     int64         // Maximum number of keys
	TTL         time.Duration // Time to live for cache entries
	BufferItems int64         // Buffer size for ristretto

	// Loader settings
	Loader      Loader[K, V]  // Data loader interface
	LoadTimeout time.Duration // Timeout for load operations

	// Async refresh settings
	EnableAsync     bool          // Enable async refresh
	RefreshInterval time.Duration // Interval for async refresh
	RefreshMode     RefreshMode   // Single, batch, or auto refresh mode
	MaxConcurrency  int           // Max concurrent refresh operations
	BatchSize       int           // Batch size for batch refresh mode

	// Monitoring
	EnableStats bool // Enable statistics collection
}

// Flux represents the cache instance
type Flux[K Key, V any] struct {
	cache  *ristretto.Cache[K, V] // Underlying ristretto cache
	config *Config[K, V]          // Configuration
	stats  *Stats                 // Statistics

	// Async refresh control
	refreshTicker *time.Ticker
	refreshQueue  chan K
	batchQueue    chan []K      // Queue for batch refresh jobs
	allQueue      chan struct{} // Queue for LoadAll refresh jobs
	stopCh        chan struct{}
	wg            sync.WaitGroup
	closed        int32

	// Key tracking for async refresh
	trackedKeys map[K]time.Time
	keysMutex   sync.RWMutex

	// Loader capabilities (cached for performance)
	hasLoadAll   bool
	hasLoadBatch bool
	hasLoad      bool
}

// New creates a new Flux cache instance
func New[K Key, V any](opts ...Option[K, V]) (*Flux[K, V], error) {
	config := NewFluxConfig[K, V]()

	// Apply options
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create ristretto cache
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config[K, V]{
		NumCounters: config.MaxKeys * 10, // 10x capacity
		MaxCost:     config.MaxKeys,      // Max number of keys
		BufferItems: config.BufferItems,
		Metrics:     config.EnableStats,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	flux := &Flux[K, V]{
		cache:       ristrettoCache,
		config:      config,
		stats:       &Stats{},
		stopCh:      make(chan struct{}),
		trackedKeys: make(map[K]time.Time),
	}

	// Check loader capabilities
	flux.checkLoaderCapabilities()

	// Start async refresh if enabled
	if config.EnableAsync {
		flux.startAsyncRefresh()
	}

	return flux, nil
}

// NewFluxConfig creates a new Config with default values
func NewFluxConfig[K Key, V any]() *Config[K, V] {
	return &Config[K, V]{
		MaxKeys:         1000,
		TTL:             10 * time.Minute,
		BufferItems:     64,
		LoadTimeout:     30 * time.Second,
		EnableAsync:     false,
		RefreshInterval: 2 * time.Minute,
		RefreshMode:     RefreshSingle,
		MaxConcurrency:  10,
		BatchSize:       100,
		EnableStats:     true,
	}
}

// checkLoaderCapabilities examines what methods the loader implements
func (f *Flux[K, V]) checkLoaderCapabilities() {
	if f.config.Loader == nil {
		return
	}

	// Check if loader implements Load method
	f.hasLoad = true // All loaders must implement Load (it's in the base interface)

	// Check if loader implements LoadBatch method
	_, f.hasLoadBatch = f.config.Loader.(BatchLoader[K, V])

	// Check if loader implements LoadAll method
	_, f.hasLoadAll = f.config.Loader.(AllLoader[K, V])
}

// Get retrieves a value from cache or loads it if not present
func (f *Flux[K, V]) Get(ctx context.Context, key K) (V, error) {
	if atomic.LoadInt32(&f.closed) == 1 {
		return *new(V), ErrCacheClosed
	}

	// Try to get from cache first
	if value, found := f.cache.Get(key); found {
		if f.config.EnableStats {
			atomic.AddUint64(&f.stats.Hits, 1)
		}
		f.trackKey(key)
		return value, nil
	}

	// Cache miss - load from source
	if f.config.EnableStats {
		atomic.AddUint64(&f.stats.Misses, 1)
	}

	return f.loadAndSet(ctx, key)
}

// Set stores a value in the cache
func (f *Flux[K, V]) Set(ctx context.Context, key K, value V) error {
	if atomic.LoadInt32(&f.closed) == 1 {
		return ErrCacheClosed
	}

	cost := int64(1)
	if f.config.TTL > 0 {
		f.cache.SetWithTTL(key, value, cost, f.config.TTL)
	} else {
		f.cache.Set(key, value, cost)
	}

	f.trackKey(key)
	return nil
}

// Del removes a key from the cache
func (f *Flux[K, V]) Del(key K) error {
	if atomic.LoadInt32(&f.closed) == 1 {
		return ErrCacheClosed
	}

	f.cache.Del(key)
	f.untrackKey(key)
	return nil
}

// Refresh manually refreshes a specific key
func (f *Flux[K, V]) Refresh(ctx context.Context, key K) error {
	if atomic.LoadInt32(&f.closed) == 1 {
		return ErrCacheClosed
	}

	if f.config.Loader == nil {
		return ErrLoaderNotSet
	}

	_, err := f.loadAndSet(ctx, key)
	return err
}

// RefreshAll manually refreshes all data using the best available method
func (f *Flux[K, V]) RefreshAll(ctx context.Context) error {
	if atomic.LoadInt32(&f.closed) == 1 {
		return ErrCacheClosed
	}

	if f.config.Loader == nil {
		return ErrLoaderNotSet
	}

	return f.refreshAll(ctx)
}

// Keys returns all keys currently in the cache
func (f *Flux[K, V]) Keys() []K {
	if atomic.LoadInt32(&f.closed) == 1 {
		return nil
	}

	f.keysMutex.RLock()
	keys := make([]K, 0, len(f.trackedKeys))
	for key := range f.trackedKeys {
		keys = append(keys, key)
	}
	f.keysMutex.RUnlock()

	return keys
}

// Stats returns current cache statistics
func (f *Flux[K, V]) Stats() *Stats {
	if !f.config.EnableStats {
		return nil
	}

	stats := &Stats{
		Hits:          atomic.LoadUint64(&f.stats.Hits),
		Misses:        atomic.LoadUint64(&f.stats.Misses),
		Loads:         atomic.LoadUint64(&f.stats.Loads),
		LoadErrors:    atomic.LoadUint64(&f.stats.LoadErrors),
		Refreshes:     atomic.LoadUint64(&f.stats.Refreshes),
		RefreshErrors: atomic.LoadUint64(&f.stats.RefreshErrors),
	}

	// Get key count from tracked keys
	f.keysMutex.RLock()
	stats.Keys = uint64(len(f.trackedKeys))
	f.keysMutex.RUnlock()

	// Get metrics from ristretto if available
	if metrics := f.cache.Metrics; metrics != nil {
		stats.Size = uint64(metrics.CostAdded() - metrics.CostEvicted())
	}

	return stats
}

// Close shuts down the cache and stops all background operations
func (f *Flux[K, V]) Close() error {
	if !atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		return nil // Already closed
	}

	// Stop async refresh
	if f.config.EnableAsync {
		close(f.stopCh)
		if f.refreshTicker != nil {
			f.refreshTicker.Stop()
		}
		if f.refreshQueue != nil {
			close(f.refreshQueue)
		}
		if f.batchQueue != nil {
			close(f.batchQueue)
		}
		if f.allQueue != nil {
			close(f.allQueue)
		}
		f.wg.Wait()
	}

	// Close ristretto cache
	f.cache.Close()
	return nil
}

// loadAndSet loads data from source and stores it in cache
func (f *Flux[K, V]) loadAndSet(ctx context.Context, key K) (V, error) {
	if f.config.Loader == nil {
		return *new(V), ErrLoaderNotSet
	}

	// Create timeout context
	loadCtx := ctx
	if f.config.LoadTimeout > 0 {
		var cancel context.CancelFunc
		loadCtx, cancel = context.WithTimeout(ctx, f.config.LoadTimeout)
		defer cancel()
	}

	// Load data
	if f.config.EnableStats {
		atomic.AddUint64(&f.stats.Loads, 1)
	}

	value, err := f.config.Loader.Load(loadCtx, key)
	if err != nil {
		if f.config.EnableStats {
			atomic.AddUint64(&f.stats.LoadErrors, 1)
		}
		return *new(V), fmt.Errorf("failed to load key %v: %w", key, err)
	}

	// Store in cache
	cost := int64(1)
	if f.config.TTL > 0 {
		f.cache.SetWithTTL(key, value, cost, f.config.TTL)
	} else {
		f.cache.Set(key, value, cost)
	}

	f.trackKey(key)
	return value, nil
}

// refreshAll refreshes all data using LoadAll method
func (f *Flux[K, V]) refreshAll(ctx context.Context) error {
	allLoader, ok := f.config.Loader.(AllLoader[K, V])
	if !ok {
		return fmt.Errorf("%w: LoadAll method not available", ErrLoaderMethodNotImplemented)
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.config.LoadTimeout)
	defer cancel()

	allData, err := allLoader.LoadAll(ctx)
	if err != nil {
		if f.config.EnableStats {
			atomic.AddUint64(&f.stats.RefreshErrors, 1)
		}
		return fmt.Errorf("failed to load all data: %w", err)
	}

	// Update cache with all data
	f.updateCacheWithBatchValues(allData)

	if f.config.EnableStats {
		atomic.AddUint64(&f.stats.Refreshes, uint64(len(allData)))
	}

	return nil
}

// updateCacheWithBatchValues updates the cache with batch-loaded values
func (f *Flux[K, V]) updateCacheWithBatchValues(values map[K]V) {
	if len(values) == 0 {
		return
	}

	now := time.Now()
	cost := int64(1)

	f.keysMutex.Lock()
	defer f.keysMutex.Unlock()

	for key, value := range values {
		// Update cache entry
		if f.config.TTL > 0 {
			f.cache.SetWithTTL(key, value, cost, f.config.TTL)
		} else {
			f.cache.Set(key, value, cost)
		}

		// Update tracking timestamp
		f.trackedKeys[key] = now
	}
}

// trackKey adds a key to the tracking list for async refresh
func (f *Flux[K, V]) trackKey(key K) {
	if !f.config.EnableAsync {
		return
	}

	f.keysMutex.Lock()
	f.trackedKeys[key] = time.Now()
	f.keysMutex.Unlock()
}

// untrackKey removes a key from the tracking list
func (f *Flux[K, V]) untrackKey(key K) {
	if !f.config.EnableAsync {
		return
	}

	f.keysMutex.Lock()
	delete(f.trackedKeys, key)
	f.keysMutex.Unlock()
}

// startAsyncRefresh starts the async refresh background process
func (f *Flux[K, V]) startAsyncRefresh() {
	f.refreshQueue = make(chan K, f.config.MaxConcurrency)
	f.batchQueue = make(chan []K, f.config.MaxConcurrency)
	f.allQueue = make(chan struct{}, 1) // Only one LoadAll operation at a time
	f.refreshTicker = time.NewTicker(f.config.RefreshInterval)

	// Start worker goroutines for single key refresh
	for i := 0; i < f.config.MaxConcurrency; i++ {
		f.wg.Add(1)
		go f.singleWorker()
	}

	// Start worker goroutines for batch refresh
	for i := 0; i < f.config.MaxConcurrency; i++ {
		f.wg.Add(1)
		go f.batchWorker()
	}

	// Start worker for LoadAll refresh
	f.wg.Add(1)
	go f.allWorker()

	// Start scheduler goroutine
	f.wg.Add(1)
	go f.refreshScheduler()
}

// refreshScheduler schedules keys for refresh
func (f *Flux[K, V]) refreshScheduler() {
	defer f.wg.Done()

	for {
		select {
		case <-f.refreshTicker.C:
			f.scheduleRefresh()
		case <-f.stopCh:
			return
		}
	}
}

// refreshWorker processes single key refresh requests
func (f *Flux[K, V]) singleWorker() {
	defer f.wg.Done()

	for {
		select {
		case key, ok := <-f.refreshQueue:
			if !ok {
				return // Channel closed
			}
			_ = f.refreshKey(key)
		case <-f.stopCh:
			return
		}
	}
}

// batchRefreshWorker processes batch refresh requests
func (f *Flux[K, V]) batchWorker() {
	defer f.wg.Done()

	for {
		select {
		case batch, ok := <-f.batchQueue:
			if !ok {
				return // Channel closed
			}
			_ = f.refreshBatch(batch)
		case <-f.stopCh:
			return
		}
	}
}

// allWorker processes LoadAll refresh requests
func (f *Flux[K, V]) allWorker() {
	defer f.wg.Done()

	for {
		select {
		case _, ok := <-f.allQueue:
			if !ok {
				return // Channel closed
			}
			_ = f.refreshAll(context.Background())
		case <-f.stopCh:
			return
		}
	}
}

// scheduleRefresh schedules keys for refresh based on mode
func (f *Flux[K, V]) scheduleRefresh() {
	f.keysMutex.RLock()
	keysToRefresh := make([]K, 0, len(f.trackedKeys))
	now := time.Now()

	for key, lastAccess := range f.trackedKeys {
		// Check if key should be refreshed (not expired and accessed recently)
		if f.shouldRefreshKey(key, lastAccess, now) {
			keysToRefresh = append(keysToRefresh, key)
		}
	}
	f.keysMutex.RUnlock()

	if len(keysToRefresh) == 0 {
		return
	}

	switch f.config.RefreshMode {
	case RefreshSingle:
		f.scheduleSingleRefresh(keysToRefresh)
	case RefreshBatch:
		f.scheduleBatchRefresh(keysToRefresh)
	case RefreshAll:
		f.scheduleAllRefresh()
	case RefreshAuto:
		f.scheduleAutoRefresh(keysToRefresh)
	}
}

// scheduleAutoRefresh automatically selects the best refresh method
func (f *Flux[K, V]) scheduleAutoRefresh(keys []K) {
	// Priority: LoadAll > LoadBatch > Load
	if f.hasLoadAll {
		f.scheduleAllRefresh()
	} else if f.hasLoadBatch {
		f.scheduleBatchRefresh(keys)
	} else if f.hasLoad {
		f.scheduleSingleRefresh(keys)
	}
}

// shouldRefreshKey determines if a key should be refreshed
func (f *Flux[K, V]) shouldRefreshKey(key K, lastAccess, now time.Time) bool {
	// Don't refresh if key hasn't been accessed recently
	if f.config.TTL > 0 && now.Sub(lastAccess) > f.config.TTL {
		// Remove expired keys from tracking
		go f.untrackKey(key)
		return false
	}

	// Check if key still exists in cache
	if _, found := f.cache.Get(key); !found {
		// Remove non-existent keys from tracking
		go f.untrackKey(key)
		return false
	}

	return true
}

// scheduleSingleRefresh schedules individual key refreshes
func (f *Flux[K, V]) scheduleSingleRefresh(keys []K) {
	for _, key := range keys {
		select {
		case f.refreshQueue <- key:
			// Successfully queued
		default:
			// Queue full, skip this key
			return
		}
	}
}

// scheduleAllRefresh schedules LoadAll refresh
func (f *Flux[K, V]) scheduleAllRefresh() {
	select {
	case f.allQueue <- struct{}{}:
		// Successfully queued
	default:
		// LoadAll already in progress or queue full, skip
	}
}

// scheduleBatchRefresh schedules batch key refreshes with concurrency control
func (f *Flux[K, V]) scheduleBatchRefresh(keys []K) {
	if f.config.Loader == nil {
		return
	}

	// Check if loader supports batch operations
	_, supportsBatch := f.config.Loader.(BatchLoader[K, V])
	if !supportsBatch {
		// Fall back to single refresh
		f.scheduleSingleRefresh(keys)
		return
	}

	// Process keys in batches, respecting maxConcurrency
	for i := 0; i < len(keys); i += f.config.BatchSize {
		end := i + f.config.BatchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		select {
		case f.batchQueue <- batch:
			// Successfully queued batch
		case <-f.stopCh:
			return
		default:
			// Batch queue full, skip remaining batches
			// This respects the maxConcurrency limit
			return
		}
	}
}

// processBatchRefresh processes a batch of keys for refresh
func (f *Flux[K, V]) refreshBatch(keys []K) error {
	batchLoader, ok := f.config.Loader.(BatchLoader[K, V])
	if !ok {
		return fmt.Errorf("%w: LoadBatch method not available", ErrLoaderMethodNotImplemented)
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.config.LoadTimeout)
	defer cancel()

	batchData, err := batchLoader.LoadBatch(ctx, keys)
	if err != nil {
		if f.config.EnableStats {
			atomic.AddUint64(&f.stats.RefreshErrors, uint64(len(keys)))
		}
		return fmt.Errorf("failed to load batch data: %w", err)
	}

	// Update cache with loaded batchData
	f.updateCacheWithBatchValues(batchData)

	if f.config.EnableStats {
		atomic.AddUint64(&f.stats.Refreshes, uint64(len(batchData)))
	}

	return nil
}

// refreshKey refreshes a single key
func (f *Flux[K, V]) refreshKey(key K) error {
	ctx, cancel := context.WithTimeout(context.Background(), f.config.LoadTimeout)
	defer cancel()

	value, err := f.config.Loader.Load(ctx, key)
	if err != nil {
		if f.config.EnableStats {
			atomic.AddUint64(&f.stats.RefreshErrors, 1)
		}
		return fmt.Errorf("failed to load key: %w", err)
	}

	// Update cache with loaded value
	f.updateCacheWithBatchValues(map[K]V{key: value})

	if f.config.EnableStats {
		atomic.AddUint64(&f.stats.Refreshes, 1)
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig[K Key, V any](config *Config[K, V]) error {
	if config.MaxKeys <= 0 {
		return errors.New("maxKeys must be positive")
	}
	if config.LoadTimeout <= 0 {
		return errors.New("loadTimeout must be positive")
	}
	if config.EnableAsync {
		if config.RefreshInterval <= 0 {
			return errors.New("refreshInterval must be positive when async is enabled")
		}
		if config.MaxConcurrency <= 0 {
			return errors.New("maxConcurrency must be positive when async is enabled")
		}
		if config.BatchSize <= 0 {
			return errors.New("batchSize must be positive")
		}

		// Validate loader methods based on refresh mode
		if err := validateLoaderMethods(config); err != nil {
			return err
		}
	}
	return nil
}

// validateLoaderMethods validates that required loader methods are implemented
func validateLoaderMethods[K Key, V any](config *Config[K, V]) error {
	if config.Loader == nil {
		return ErrLoaderNotSet
	}

	switch config.RefreshMode {
	case RefreshSingle:
		// RefreshSingle only needs Load method (which is guaranteed by the interface)
		return nil

	case RefreshBatch:
		// RefreshBatch requires LoadBatch method
		if _, ok := config.Loader.(BatchLoader[K, V]); !ok {
			return fmt.Errorf("%w: LoadBatch method required for RefreshBatch mode", ErrLoaderMethodNotImplemented)
		}
		return nil
	case RefreshAll:
		// RefreshAll requires LoadAll method
		if _, ok := config.Loader.(AllLoader[K, V]); !ok {
			return fmt.Errorf("%w: LoadAll method required for RefreshAll mode", ErrLoaderMethodNotImplemented)
		}
		return nil

	case RefreshAuto:
		// RefreshAuto needs at least one method (Load is guaranteed, so always valid)
		return nil

	default:
		return fmt.Errorf("unknown refresh mode: %d", config.RefreshMode)
	}
}

// GetLoaderCapabilities returns information about what loader methods are available
func (f *Flux[K, V]) GetLoaderCapabilities() map[string]bool {
	return map[string]bool{
		"Load":      f.hasLoad,
		"LoadBatch": f.hasLoadBatch,
		"LoadAll":   f.hasLoadAll,
	}
}
