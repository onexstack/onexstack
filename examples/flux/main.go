package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/maypok86/otter/v2"
	"github.com/onexstack/onexstack/pkg/flux"
)

// User 示例数据结构
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Product 示例数据结构
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// =============================================================================
// 1. 基础 Loader 实现（单个加载）
// =============================================================================

type UserLoader struct {
	// 模拟数据库或外部服务
	users map[string]*User
}

func NewUserLoader() *UserLoader {
	return &UserLoader{
		users: map[string]*User{
			"1":  {ID: 1, Name: "Alice", Email: "alice@example.com"},
			"2":  {ID: 2, Name: "Bob", Email: "bob@example.com"},
			"3":  {ID: 3, Name: "Charlie", Email: "charlie@example.com"},
			"4":  {ID: 4, Name: "Charlie", Email: "charlie@example.com"},
			"5":  {ID: 5, Name: "Charlie", Email: "charlie@example.com"},
			"6":  {ID: 6, Name: "Charlie", Email: "charlie@example.com"},
			"7":  {ID: 7, Name: "Charlie", Email: "charlie@example.com"},
			"8":  {ID: 8, Name: "Charlie", Email: "charlie@example.com"},
			"9":  {ID: 9, Name: "Charlie", Email: "charlie@example.com"},
			"10": {ID: 10, Name: "Charlie", Email: "charlie@example.com"},
			"11": {ID: 11, Name: "Charlie", Email: "charlie@example.com"},
		},
	}
}

// Load 实现单个用户加载
func (l *UserLoader) Load(ctx context.Context, key string) (*User, error) {
	fmt.Printf("Loading user with key: %s\n", key)

	// 模拟加载延迟
	time.Sleep(100 * time.Millisecond)

	user, exists := l.users[key]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", key)
	}
	user.Name = user.Name + time.Now().UTC().String()

	return user, nil
}

// =============================================================================
// 2. 批量 Loader 实现（支持批量加载）
// =============================================================================

type ProductBatchLoader struct {
	products map[int]*Product
}

func NewProductBatchLoader() *ProductBatchLoader {
	return &ProductBatchLoader{
		products: map[int]*Product{
			1:  {ID: 1, Name: "Laptop", Price: 999.99},
			2:  {ID: 2, Name: "Mouse", Price: 29.99},
			3:  {ID: 3, Name: "Keyboard", Price: 79.99},
			4:  {ID: 4, Name: "Monitor", Price: 299.99},
			5:  {ID: 5, Name: "Headphones", Price: 149.99},
			6:  {ID: 6, Name: "Headphones", Price: 149.99},
			7:  {ID: 7, Name: "Headphones", Price: 149.99},
			8:  {ID: 8, Name: "Headphones", Price: 149.99},
			9:  {ID: 9, Name: "Headphones", Price: 149.99},
			10: {ID: 10, Name: "Headphones", Price: 149.99},
			11: {ID: 11, Name: "Headphones", Price: 149.99},
			12: {ID: 12, Name: "Headphones", Price: 149.99},
			13: {ID: 13, Name: "Headphones", Price: 149.99},
			14: {ID: 14, Name: "Headphones", Price: 149.99},
		},
	}
}

// Load 实现单个产品加载
func (l *ProductBatchLoader) Load(ctx context.Context, key int) (*Product, error) {
	fmt.Printf("Loading product with key: %d\n", key)

	time.Sleep(50 * time.Millisecond)

	product, exists := l.products[key]
	if !exists {
		return nil, fmt.Errorf("product not found: %d", key)
	}

	return product, nil
}

// LoadBatch 实现批量产品加载
func (l *ProductBatchLoader) LoadBatch(ctx context.Context, keys []int) (map[int]*Product, error) {
	fmt.Printf("Batch loading products with keys: %v\n", keys)

	// 模拟批量加载延迟（通常比单个加载更高效）
	time.Sleep(500 * time.Millisecond)

	result := make(map[int]*Product)
	for _, key := range keys {
		if product, exists := l.products[key]; exists {
			result[key] = product
		}
	}

	return result, nil
}

// =============================================================================
// 3. 全量 Loader 实现（支持加载所有数据）
// =============================================================================

type ConfigLoader struct {
	configs map[string]string
}

func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		configs: map[string]string{
			"app.name":    "MyApp",
			"app.version": "1.0.0",
			"db.host":     "localhost",
			"db.port":     "5432",
			"redis.host":  "localhost",
			"redis.port":  "6379",
		},
	}
}

// Load 实现单个配置加载
func (l *ConfigLoader) Load(ctx context.Context, key string) (string, error) {
	fmt.Printf("Loading config with key: %s\n", key)

	time.Sleep(30 * time.Millisecond)

	value, exists := l.configs[key]
	if !exists {
		return "", fmt.Errorf("config not found: %s", key)
	}

	return value, nil
}

// LoadBatch 实现批量配置加载
func (l *ConfigLoader) LoadBatch(ctx context.Context, keys []string) (map[string]string, error) {
	fmt.Printf("Batch loading configs with keys: %v\n", keys)

	time.Sleep(100 * time.Millisecond)

	result := make(map[string]string)
	for _, key := range keys {
		if value, exists := l.configs[key]; exists {
			result[key] = value
		}
	}

	return result, nil
}

// LoadAll 实现全量配置加载
func (l *ConfigLoader) LoadAll(ctx context.Context) (map[string]string, error) {
	fmt.Printf("Loading all configs\n")

	time.Sleep(150 * time.Millisecond)

	// 返回所有配置的副本
	result := make(map[string]string)
	for k, v := range l.configs {
		result[k] = v
	}

	return result, nil
}

// =============================================================================
// 示例函数
// =============================================================================

// 示例1：基础用法 - 简单的缓存
func example1BasicUsage() {
	fmt.Println("\n=== Example 1: Basic Usage ===")

	userLoader := NewUserLoader()

	// 创建用户缓存
	userCache, err := flux.NewFlux[string, *User](
		flux.WithMaxKeys[string, *User](100),
		flux.WithTTL[string, *User](5*time.Minute),
		flux.WithLoader[string, *User](userLoader),
		flux.WithLoadTimeout[string, *User](5*time.Second),
		flux.WithEnableAsync[string, *User](true),
	)
	if err != nil {
		log.Fatalf("Failed to create user cache: %v", err)
	}
	defer userCache.Close()

	ctx := context.Background()

	// 第一次获取 - 会从 loader 加载
	user1, err := userCache.Get(ctx, "1")
	if err != nil {
		log.Printf("Error getting user 1: %v", err)
		return
	}
	fmt.Printf("Got user: %+v\n", user1)
	time.Sleep(10 * time.Second)

	// 第二次获取 - 从缓存返回
	user1Again, err := userCache.Get(ctx, "1")
	if err != nil {
		log.Printf("Error getting user 1 again: %v", err)
		return
	}
	fmt.Printf("Got user from cache: %+v\n", user1Again)

	// 手动设置缓存
	newUser := &User{ID: 999, Name: "New User", Email: "new@example.com"}
	err = userCache.Set(ctx, "999", newUser)
	if err != nil {
		log.Printf("Error setting user: %v", err)
	}

	// 获取统计信息
	stats := userCache.Stats()
	fmt.Printf("Cache stats: %+v\n", stats)
}

// 示例2：批量加载缓存
func example2BatchLoading() {
	fmt.Println("\n=== Example 2: Batch Loading ===")

	productLoader := NewProductBatchLoader()

	// 创建产品缓存，启用批量刷新
	productCache, err := flux.NewFlux[int, *Product](
		flux.WithMaxKeys[int, *Product](1000),
		flux.WithTTL[int, *Product](10*time.Minute),
		flux.WithLoader[int, *Product](productLoader),
		flux.WithAsyncRefresh[int, *Product](2*time.Second),    // 启用异步刷新
		flux.WithLoadTimeout[int, *Product](3*time.Second),     // 启用异步刷新
		flux.WithRefreshMode[int, *Product](flux.RefreshBatch), // 批量刷新模式
		flux.WithBatchSize[int, *Product](2),
		flux.WithMaxConcurrency[int, *Product](2),
	)
	if err != nil {
		log.Fatalf("Failed to create product cache: %v", err)
	}
	defer productCache.Close()

	ctx := context.Background()

	// 获取多个产品
	productIDs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
	for _, id := range productIDs {
		product, err := productCache.Get(ctx, id)
		if err != nil {
			log.Printf("Error getting product %d: %v", id, err)
			continue
		}
		fmt.Printf("Got product: %+v\n", product)
	}

	// 等待一会儿，让异步刷新工作
	fmt.Println("Waiting for async refresh...")
	time.Sleep(4 * time.Second)

	// 显示加载器能力
	capabilities := productCache.GetLoaderCapabilities()
	fmt.Printf("Loader capabilities: %+v\n", capabilities)
}

// 示例3：全量加载缓存（配置缓存）
func example3FullLoading() {
	fmt.Println("\n=== Example 3: Full Loading (Config Cache) ===")

	configLoader := NewConfigLoader()

	// 创建配置缓存，使用自动刷新模式
	configCache, err := flux.NewFlux[string, string](
		flux.WithMaxKeys[string, string](50),
		flux.WithTTL[string, string](0), // 永不过期
		flux.WithLoader[string, string](configLoader),
		flux.WithAsyncRefresh[string, string](1*time.Minute),
		flux.WithRefreshMode[string, string](flux.RefreshAuto), // 自动选择最佳刷新方式
	)
	if err != nil {
		log.Fatalf("Failed to create config cache: %v", err)
	}
	defer configCache.Close()

	ctx := context.Background()

	// 获取配置
	configs := []string{"app.name", "app.version", "db.host", "redis.port"}
	for _, key := range configs {
		value, err := configCache.Get(ctx, key)
		if err != nil {
			log.Printf("Error getting config %s: %v", key, err)
			continue
		}
		fmt.Printf("Config %s: %s\n", key, value)
	}

	// 显示所有缓存的键
	allKeys := configCache.Keys()
	fmt.Printf("All cached keys: %v\n", allKeys)
}

// 示例4：使用配置对象
func example4ConfigObject() {
	fmt.Println("\n=== Example 4: Using Config Object ===")

	userLoader := NewUserLoader()

	// 方式1：使用 NewFluxConfig 创建默认配置后修改
	config := flux.NewFluxFluxConfig[string, *User]()
	config.MaxKeys = 500
	config.TTL = 15 * time.Minute
	config.Loader = userLoader
	config.LoadTimeout = 5 * time.Second
	config.EnableAsync = true
	config.RefreshInterval = 2 * time.Minute
	config.RefreshMode = flux.RefreshSingle

	userCache, err := flux.NewFlux(flux.WithConfig(config))
	if err != nil {
		log.Fatalf("Failed to create user cache with config: %v", err)
	}
	defer userCache.Close()

	ctx := context.Background()

	// 测试缓存
	user, err := userCache.Get(ctx, "2")
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}
	fmt.Printf("Got user from config-based cache: %+v\n", user)
}

// 示例5：错误处理和边界情况
func example5ErrorHandling() {
	fmt.Println("\n=== Example 5: Error Handling ===")

	userLoader := NewUserLoader()

	userCache, err := flux.NewFlux[string, *User](
		flux.WithMaxKeys[string, *User](10),
		flux.WithTTL[string, *User](1*time.Second), // 很短的TTL用于测试
		flux.WithLoader[string, *User](userLoader),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer userCache.Close()

	ctx := context.Background()

	// 测试不存在的键
	_, err = userCache.Get(ctx, "999")
	if err != nil {
		fmt.Printf("Expected error for non-existent key: %v\n", err)
	}

	// 测试超时
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	_, err = userCache.Get(timeoutCtx, "1")
	if err != nil {
		fmt.Printf("Expected timeout error: %v\n", err)
	}

	// 测试缓存过期
	user, _ := userCache.Get(ctx, "1")
	fmt.Printf("Got user: %+v\n", user)

	fmt.Println("Waiting for cache to expire...")
	time.Sleep(2 * time.Second)

	// 过期后重新获取
	user, _ = userCache.Get(ctx, "1")
	fmt.Printf("Got user after expiry: %+v\n", user)
}

// 示例6：性能测试
func example6Performance() {
	fmt.Println("\n=== Example 6: Performance Test ===")

	userLoader := NewUserLoader()

	userCache, err := flux.NewFlux[string, *User](
		flux.WithMaxKeys[string, *User](1000),
		flux.WithTTL[string, *User](10*time.Minute),
		flux.WithLoader[string, *User](userLoader),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer userCache.Close()

	ctx := context.Background()

	// 性能测试
	const numRequests = 1000
	keys := []string{"1", "2", "3"}

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		key := keys[i%len(keys)]
		_, err := userCache.Get(ctx, key)
		if err != nil {
			log.Printf("Error in performance test: %v", err)
		}
	}

	duration := time.Since(start)
	fmt.Printf("Completed %d requests in %v\n", numRequests, duration)
	fmt.Printf("Average: %v per request\n", duration/numRequests)

	// 显示最终统计
	stats := userCache.Stats()
	fmt.Printf("Final stats: %+v\n", stats)
	fmt.Printf("Hit ratio: %.2f%%\n", float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)
}

// 示例：Otter 包性能对比测试
func exampleOtterPerformanceComparison() {
	fmt.Println("\n=== Otter vs Flux Performance Comparison ===")

	userLoader := NewUserLoader()
	ctx := context.Background()

	// 1. 创建 Flux 缓存
	fluxCache, err := flux.NewFlux[string, *User](flux.WithMaxKeys[string, *User](1000),
		flux.WithTTL[string, *User](10*time.Minute),
		flux.WithLoader[string, *User](userLoader),
	)
	if err != nil {
		log.Fatalf("Failed to create flux cache: %v", err)
	}
	defer fluxCache.Close()

	// 2. 创建 Otter 缓存
	otterCache := otter.Must[string, *User](&otter.Options[string, *User]{
		MaximumSize:       10_000,
		InitialCapacity:   500,
		ExpiryCalculator:  otter.ExpiryAccessing[string, *User](10 * time.Minute),      // Reset timer on reads/writes
		RefreshCalculator: otter.RefreshWriting[string, *User](500 * time.Millisecond), // Refresh after writes
		// StatsRecorder:     counter,                                                      // Attach stats collector
	})

	const numRequests = 1000
	keys := []string{"1", "2", "3", "4", "5"}

	// 3. 测试 Flux 性能
	fmt.Println("\n--- Testing Flux Cache ---")
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		key := keys[i%len(keys)]
		_, err := fluxCache.Get(ctx, key)
		if err != nil {
			log.Printf("Flux error: %v", err)
		}
	}

	fluxDuration := time.Since(start)
	fluxStats := fluxCache.Stats()

	fmt.Printf("Flux - Completed %d requests in %v\n", numRequests, fluxDuration)
	fmt.Printf("Flux - Average: %v per request\n", fluxDuration/numRequests)
	fmt.Printf("Flux - Stats: %+v\n", fluxStats)
	fmt.Printf("Flux - Hit ratio: %.2f%%\n",
		float64(fluxStats.Hits)/float64(fluxStats.Hits+fluxStats.Misses)*100)

	// 4. 测试 Otter 性能
	fmt.Println("\n--- Testing Otter Cache ---")

	// 模拟 loader 函数
	loadFunc := func(key string) (*User, error) {
		return userLoader.Load(ctx, key)
	}

	start = time.Now()
	var otterHits, otterMisses int64

	for i := 0; i < numRequests; i++ {
		key := keys[i%len(keys)]

		// 先尝试获取
		if user, ok := otterCache.GetIfPresent(key); ok {
			atomic.AddInt64(&otterHits, 1)
			_ = user
		} else {
			// 缓存未命中，加载数据
			atomic.AddInt64(&otterMisses, 1)
			user, err := loadFunc(key)
			if err != nil {
				log.Printf("Otter load error: %v", err)
				continue
			}
			otterCache.Set(key, user)
		}
	}

	otterDuration := time.Since(start)

	fmt.Printf("Otter - Completed %d requests in %v\n", numRequests, otterDuration)
	fmt.Printf("Otter - Average: %v per request\n", otterDuration/numRequests)
	fmt.Printf("Otter - Hits: %d, Misses: %d\n", otterHits, otterMisses)
	fmt.Printf("Otter - Hit ratio: %.2f%%\n",
		float64(otterHits)/float64(otterHits+otterMisses)*100)

	// 5. 性能对比结果
	fmt.Println("\n--- Performance Comparison ---")
	fmt.Printf("Flux duration:  %v\n", fluxDuration)
	fmt.Printf("Otter duration: %v\n", otterDuration)

	if fluxDuration < otterDuration {
		improvement := float64(otterDuration-fluxDuration) / float64(otterDuration) * 100
		fmt.Printf("������ Flux is %.2f%% faster than Otter\n", improvement)
	} else {
		degradation := float64(fluxDuration-otterDuration) / float64(fluxDuration) * 100
		fmt.Printf("⚠️  Flux is %.2f%% slower than Otter\n", degradation)
	}

	// 6. 内存使用情况对比
	fmt.Println("\n--- Memory Usage ---")
	var m1, m2 runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&m1)
	fmt.Printf("Memory before cleanup: %d KB\n", m1.Alloc/1024)

	// 清理缓存
	fluxCache.Close()

	runtime.GC()
	runtime.ReadMemStats(&m2)
	fmt.Printf("Memory after cleanup:  %d KB\n", m2.Alloc/1024)
}

// 高并发性能测试
func exampleConcurrentPerformanceComparison() {
	fmt.Println("\n=== Concurrent Performance Comparison ===")

	userLoader := NewUserLoader()
	ctx := context.Background()

	// 创建缓存
	fluxCache, _ := flux.NewFlux[string, *User](flux.WithMaxKeys[string, *User](1000),
		flux.WithTTL[string, *User](10*time.Minute),
		flux.WithLoader[string, *User](userLoader),
	)
	defer fluxCache.Close()

	otterCache := otter.Must[string, *User](&otter.Options[string, *User]{
		MaximumSize:       10_000,
		InitialCapacity:   500,
		ExpiryCalculator:  otter.ExpiryAccessing[string, *User](10 * time.Minute),      // Reset timer on reads/writes
		RefreshCalculator: otter.RefreshWriting[string, *User](500 * time.Millisecond), // Refresh after writes
		// StatsRecorder:     counter,                                                      // Attach stats collector
	})

	const numGoroutines = 100
	const requestsPerGoroutine = 100
	keys := []string{"1", "2", "3", "4", "5"}

	// 测试 Flux 并发性能
	fmt.Println("Testing Flux concurrent performance...")
	start := time.Now()
	var fluxWg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		fluxWg.Add(1)
		go func() {
			defer fluxWg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				key := keys[j%len(keys)]
				_, _ = fluxCache.Get(ctx, key)
			}
		}()
	}
	fluxWg.Wait()
	fluxConcurrentDuration := time.Since(start)

	// 测试 Otter 并发性能
	fmt.Println("Testing Otter concurrent performance...")
	loadFunc := func(key string) (*User, error) {
		return userLoader.Load(ctx, key)
	}

	start = time.Now()
	var otterWg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		otterWg.Add(1)
		go func() {
			defer otterWg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				key := keys[j%len(keys)]
				if user, ok := otterCache.GetIfPresent(key); !ok {
					if user, err := loadFunc(key); err == nil {
						otterCache.Set(key, user)
					}
				} else {
					_ = user
				}
			}
		}()
	}
	otterWg.Wait()
	otterConcurrentDuration := time.Since(start)

	// 并发测试结果
	totalRequests := numGoroutines * requestsPerGoroutine
	fmt.Printf("\n--- Concurrent Test Results ---\n")
	fmt.Printf("Total requests: %d (across %d goroutines)\n", totalRequests, numGoroutines)
	fmt.Printf("Flux concurrent:  %v (%v per request)\n",
		fluxConcurrentDuration, fluxConcurrentDuration/time.Duration(totalRequests))
	fmt.Printf("Otter concurrent: %v (%v per request)\n",
		otterConcurrentDuration, otterConcurrentDuration/time.Duration(totalRequests))

	if fluxConcurrentDuration < otterConcurrentDuration {
		improvement := float64(otterConcurrentDuration-fluxConcurrentDuration) /
			float64(otterConcurrentDuration) * 100
		fmt.Printf("��� Flux i2f%% faster in concurrent scenarios\n", improvement)
	} else {
		degradation := float64(fluxConcurrentDuration-otterConcurrentDuration) /
			float64(fluxConcurrentDuration) * 100
		fmt.Printf("Flux is %.2f%% slower in concurrent scenarios\n", degradation)
	}
}

// 示例：Otter、Ristretto vs Flux 性能对比测试
func exampleCachePerformanceComparison() {
	fmt.Println("\n=== Flux vs Otter vs Ristretto Performance Comparison ===")

	userLoader := NewUserLoader()
	ctx := context.Background()

	// 1. 创建 Flux 缓存
	fluxCache, err := flux.NewFlux[string, *User](
		flux.WithMaxKeys[string, *User](1e4),
		flux.WithTTL[string, *User](10*time.Minute),
		flux.WithLoader[string, *User](userLoader),
		flux.WithBufferItems[string, *User](300),
		flux.WithEnableAsync[string, *User](false),
	)
	if err != nil {
		log.Fatalf("Failed to create flux cache: %v", err)
	}
	defer fluxCache.Close()

	// 2. 创建 Otter 缓存
	otterCache := otter.Must[string, *User](&otter.Options[string, *User]{
		MaximumSize:       10_000,
		InitialCapacity:   500,
		ExpiryCalculator:  otter.ExpiryAccessing[string, *User](10 * time.Minute),      // Reset timer on reads/writes
		RefreshCalculator: otter.RefreshWriting[string, *User](500 * time.Millisecond), // Refresh after writes
		// StatsRecorder:     counter,                                                      // Attach stats collector
	})

	// 3. 创建 Ristretto 缓存 (v2 泛型版本)
	ristrettoCache, err := ristretto.NewCache[string, *User](&ristretto.Config[string, *User]{
		NumCounters: 1e4,  // 10,000个计数器
		MaxCost:     1000, // 最大成本1000
		BufferItems: 64,   // 缓冲区大小
		Metrics:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer ristrettoCache.Close()

	const numRequests = 5000
	keys := []string{"1", "2", "3", "4", "5"}

	// 测试函数
	testFlux := func() (time.Duration, *flux.Stats) {
		start := time.Now()
		for i := 0; i < numRequests; i++ {
			key := keys[i%len(keys)]
			_, err := fluxCache.Get(ctx, key)
			if err != nil {
				log.Printf("Flux error: %v", err)
			}
		}
		return time.Since(start), fluxCache.Stats()
	}

	testOtter := func() (time.Duration, int64, int64) {
		var hits, misses int64
		start := time.Now()

		for i := 0; i < numRequests; i++ {
			key := keys[i%len(keys)]

			if user, ok := otterCache.GetIfPresent(key); ok {
				atomic.AddInt64(&hits, 1)
				_ = user
			} else {
				atomic.AddInt64(&misses, 1)
				user, err := userLoader.Load(ctx, key)
				if err != nil {
					log.Printf("Otter load error: %v", err)
					continue
				}
				otterCache.Set(key, user)
			}
		}

		return time.Since(start), hits, misses
	}

	testRistretto := func() (time.Duration, int64, int64) {
		var hits, misses int64
		start := time.Now()

		for i := 0; i < numRequests; i++ {
			key := keys[i%len(keys)]

			if user, ok := ristrettoCache.Get(key); ok {
				atomic.AddInt64(&hits, 1)
				_ = user
			} else {
				atomic.AddInt64(&misses, 1)
				user, err := userLoader.Load(ctx, key)
				if err != nil {
					log.Printf("Ristretto load error: %v", err)
					continue
				}
				ristrettoCache.SetWithTTL(key, user, 1, 10*time.Minute)
				ristrettoCache.Wait() // 等待异步设置完成
			}
		}

		return time.Since(start), hits, misses
	}

	// 4. 预热所有缓存
	fmt.Println("Warming up caches...")
	for _, key := range keys {
		fluxCache.Get(ctx, key)

		if user, err := userLoader.Load(ctx, key); err == nil {
			otterCache.Set(key, user)
			ristrettoCache.Set(key, user, 1)
		}
	}
	ristrettoCache.Wait()
	time.Sleep(100 * time.Millisecond) // 确保缓存预热完成

	// 5. 执行性能测试
	fmt.Println("\n--- Performance Test Results ---")

	// 测试 Flux
	fluxDuration, fluxStats := testFlux()
	fmt.Printf("Flux:\n")
	fmt.Printf("  Duration: %v\n", fluxDuration)
	fmt.Printf("  Average:  %v per request\n", fluxDuration/numRequests)
	fmt.Printf("  Stats:    Hits=%d, Misses=%d\n", fluxStats.Hits, fluxStats.Misses)
	fmt.Printf("  Hit Rate: %.2f%%\n",
		float64(fluxStats.Hits)/float64(fluxStats.Hits+fluxStats.Misses)*100)

	// 测试 Otter
	otterDuration, otterHits, otterMisses := testOtter()
	fmt.Printf("\nOtter:\n")
	fmt.Printf("  Duration: %v\n", otterDuration)
	fmt.Printf("  Average:  %v per request\n", otterDuration/numRequests)
	fmt.Printf("  Stats:    Hits=%d, Misses=%d\n", otterHits, otterMisses)
	fmt.Printf("  Hit Rate: %.2f%%\n",
		float64(otterHits)/float64(otterHits+otterMisses)*100)

	// 测试 Ristretto
	ristrettoDuration, ristrettoHits, ristrettoMisses := testRistretto()
	fmt.Printf("\nRistretto:\n")
	fmt.Printf("  Duration: %v\n", ristrettoDuration)
	fmt.Printf("  Average:  %v per request\n", ristrettoDuration/numRequests)
	fmt.Printf("  Stats:    Hits=%d, Misses=%d\n", ristrettoHits, ristrettoMisses)
	fmt.Printf("  Hit Rate: %.2f%%\n",
		float64(ristrettoHits)/float64(ristrettoHits+ristrettoMisses)*100)

	// 6. 性能排名
	fmt.Println("\n--- Performance Ranking ---")
	type result struct {
		name     string
		duration time.Duration
	}

	results := []result{
		{"Flux", fluxDuration},
		{"Otter", otterDuration},
		{"Ristretto", ristrettoDuration},
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].duration < results[j].duration
	})

	for i, r := range results {
		var symbol string
		switch i {
		case 0:
			symbol = ""
		case 1:
			symbol = ""
		case 2:
			symbol = ""
		}
		fmt.Printf("%s %s: %v\n", symbol, r.name, r.duration)
	}

	// 7. 相对性能对比
	fastest := results[0].duration
	fmt.Println("\n--- Relative Performance ---")
	for _, r := range results {
		if r.duration == fastest {
			fmt.Printf("%s: baseline (fastest)\n", r.name)
		} else {
			slower := float64(r.duration-fastest) / float64(fastest) * 100
			fmt.Printf("%s: %.2f%% slower\n", r.name, slower)
		}
	}
}

// 高并发性能测试
func exampleConcurrentCacheComparison() {
	fmt.Println("\n=== Concurrent Performance Comparison ===")

	userLoader := NewUserLoader()
	ctx := context.Background()

	// 创建缓存
	fluxCache, _ := flux.NewFlux[string, *User](flux.WithMaxKeys[string, *User](1000),
		flux.WithTTL[string, *User](10*time.Minute),
		flux.WithLoader[string, *User](userLoader),
	)
	defer fluxCache.Close()

	otterCache := otter.Must[string, *User](&otter.Options[string, *User]{
		MaximumSize:       10_000,
		InitialCapacity:   500,
		ExpiryCalculator:  otter.ExpiryAccessing[string, *User](10 * time.Minute),      // Reset timer on reads/writes
		RefreshCalculator: otter.RefreshWriting[string, *User](500 * time.Millisecond), // Refresh after writes
		// StatsRecorder:     counter,                                                      // Attach stats collector
	})

	// 3. 创建 Ristretto 缓存 (v2 泛型版本)
	ristrettoCache, _ := ristretto.NewCache[string, *User](&ristretto.Config[string, *User]{
		NumCounters: 1e4,  // 10,000个计数器
		MaxCost:     1000, // 最大成本1000
		BufferItems: 64,   // 缓冲区大小
	})
	defer ristrettoCache.Close()

	const numGoroutines = 50
	const requestsPerGoroutine = 100
	keys := []string{"1", "2", "3", "4", "5"}

	// 预热
	for _, key := range keys {
		fluxCache.Get(ctx, key)
		if user, err := userLoader.Load(ctx, key); err == nil {
			otterCache.Set(key, user)
			ristrettoCache.Set(key, user, 1)
		}
	}
	ristrettoCache.Wait()

	// 并发测试函数
	testConcurrent := func(name string, testFunc func()) time.Duration {
		fmt.Printf("Testing %s concurrent performance...\n", name)
		start := time.Now()

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				testFunc()
			}()
		}
		wg.Wait()

		return time.Since(start)
	}

	// Flux 并发测试
	fluxDuration := testConcurrent("Flux", func() {
		for i := 0; i < requestsPerGoroutine; i++ {
			key := keys[i%len(keys)]
			_, _ = fluxCache.Get(ctx, key)
		}
	})

	// Otter 并发测试
	otterDuration := testConcurrent("Otter", func() {
		for i := 0; i < requestsPerGoroutine; i++ {
			key := keys[i%len(keys)]
			if user, ok := otterCache.GetIfPresent(key); !ok {
				if user, err := userLoader.Load(ctx, key); err == nil {
					otterCache.Set(key, user)
				}
			} else {
				_ = user
			}
		}
	})

	// Ristretto 并发测试
	ristrettoDuration := testConcurrent("Ristretto", func() {
		for i := 0; i < requestsPerGoroutine; i++ {
			key := keys[i%len(keys)]
			if _, ok := ristrettoCache.Get(key); !ok {
				if user, err := userLoader.Load(ctx, key); err == nil {
					ristrettoCache.Set(key, user, 1)
				}
			}
		}
	})

	// 并发测试结果
	totalRequests := numGoroutines * requestsPerGoroutine
	fmt.Printf("\n--- Concurrent Test Results ---\n")
	fmt.Printf("Total requests: %d (across %d goroutines)\n", totalRequests, numGoroutines)
	fmt.Printf("Flux:      %v (%v per request)\n",
		fluxDuration, fluxDuration/time.Duration(totalRequests))
	fmt.Printf("Otter:     %v (%v per request)\n",
		otterDuration, otterDuration/time.Duration(totalRequests))
	fmt.Printf("Ristretto: %v (%v per request)\n",
		ristrettoDuration, ristrettoDuration/time.Duration(totalRequests))

	// 并发性能排名
	concurrentResults := []struct {
		name     string
		duration time.Duration
	}{
		{"Flux", fluxDuration},
		{"Otter", otterDuration},
		{"Ristretto", ristrettoDuration},
	}

	sort.Slice(concurrentResults, func(i, j int) bool {
		return concurrentResults[i].duration < concurrentResults[j].duration
	})

	fmt.Println("\n--- Concurrent Performance Ranking ---")
	for i, r := range concurrentResults {
		var symbol string
		switch i {
		case 0:
			symbol = ""
		case 1:
			symbol = ""
		case 2:
			symbol = ""
		}
		fmt.Printf("%s %s: %v\n", symbol, r.name, r.duration)
	}
}

// 内存使用对比
func exampleMemoryComparison() {
	fmt.Println("\n=== Memory Usage Comparison ===")

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 这里可以创建大量缓存项来测试内存使用
	// 具体实现根据需要添加...

	runtime.GC()
	runtime.ReadMemStats(&m2)

	fmt.Printf("Memory usage: %d KB\n", (m2.Alloc-m1.Alloc)/1024)
}

func main() {
	// 运行所有示例
	example1BasicUsage()
	// example2BatchLoading()
	// example3FullLoading()
	// example4ConfigObject()
	// example5ErrorHandling()
	// example6Performance()
	// exampleOtterPerformanceComparison()
	// exampleConcurrentPerformanceComparison()
	// exampleCachePerformanceComparison()
	// exampleConcurrentCacheComparison()

	fmt.Println("\n=== All examples completed ===")
}
