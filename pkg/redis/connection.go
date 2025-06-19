package redis

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// global client instance
	defaultClient *Client
	defaultOnce   sync.Once
	clientsMap    = make(map[string]*Client)
	clientsMutex  sync.RWMutex
)

// InitDefault initializes the default Redis client with the given configuration
func InitDefault(config *Config) {
	defaultOnce.Do(func() {
		defaultClient = New(config)

		// Start a background goroutine to periodically check the connection
		go monitorConnection(defaultClient)
	})
}

// GetDefault returns the default Redis client instance
func GetDefault() *Client {
	if defaultClient == nil {
		panic("Default Redis client not initialized. Call InitDefault first.")
	}
	return defaultClient
}

// GetOrCreate returns an existing client by name or creates a new one if it doesn't exist
func GetOrCreate(name string, config *Config) *Client {
	clientsMutex.RLock()
	client, exists := clientsMap[name]
	clientsMutex.RUnlock()

	if exists {
		return client
	}

	// Lock for writing
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	// Double-check in case another goroutine created it while we were waiting
	client, exists = clientsMap[name]
	if exists {
		return client
	}

	// Create a new client
	client = New(config)
	clientsMap[name] = client

	// Start monitoring
	go monitorConnection(client)

	return client
}

// GetNamed returns a named Redis client
func GetNamed(name string) (*Client, bool) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	client, exists := clientsMap[name]
	return client, exists
}

// CloseAll closes all Redis clients
func CloseAll() {
	// Close default client if it exists
	if defaultClient != nil {
		defaultClient.Close()
	}

	// Close all named clients
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for _, client := range clientsMap {
		client.Close()
	}

	// Clear the map
	clientsMap = make(map[string]*Client)
}

// monitorConnection periodically checks the Redis connection and logs issues
func monitorConnection(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := client.Ping(ctx)
		cancel()

		if err != nil {
			log.Printf("Redis health check failed: %v", err)
		}
	}
}

// RedisClient defines the interface for Redis operations
// Useful for mocking in tests
type RedisClient interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, key string) (string, error)
	GetJSON(ctx context.Context, key string, result interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) (bool, error)
	DeleteMany(ctx context.Context, keys ...string) (int64, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	Pipeline(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error)

	// Locking methods
	AcquireLock(ctx context.Context, lockName string, expiration time.Duration, retryCount int, retryDelay time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, lockName string) (bool, error)
	ExtendLock(ctx context.Context, lockName string, additionalTime time.Duration) (bool, error)
	IsLocked(ctx context.Context, lockName string) (bool, error)

	// Set operations
	SAdd(ctx context.Context, key string, members ...interface{}) (int64, error)
	SRem(ctx context.Context, key string, members ...interface{}) (int64, error)
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)

	// Advanced patterns
	GetWithFallback(ctx context.Context, key string, fallback func(ctx context.Context) (interface{}, error), expiration time.Duration) (interface{}, error)

	// Cleanup
	Close() error
}

// Ensure Client implements RedisClient interface
var _ RedisClient = (*Client)(nil)
