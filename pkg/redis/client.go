package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// DefaultCacheExpiry is the default expiration time for cached items
	DefaultCacheExpiry = 3600 // 1 hours in seconds
)

// Client represents a Redis client with improved connection management
type Client struct {
	client        *redis.Client
	errorCount    int32
	lastErrorTime int64
	mu            sync.Mutex
	locks         map[string]string // Track acquired locks by instance
	scripts       map[string]*redis.Script
	config        *Config
}

// Config holds Redis client configuration
type Config struct {
	Host                string
	Port                int
	DB                  int
	Password            string
	MaxConnections      int
	ConnTimeout         time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	HealthCheckInterval time.Duration
}

// DefaultConfig returns default Redis configuration
func DefaultConfig() *Config {
	return &Config{
		Host:                "localhost",
		Port:                6379,
		DB:                  0,
		Password:            "",
		MaxConnections:      100,
		ConnTimeout:         2 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 15 * time.Second,
	}
}

// New creates a new Redis client with the given configuration
func New(config *Config) *Client {
	client := &Client{
		locks:   make(map[string]string),
		scripts: make(map[string]*redis.Script),
		config:  config,
	}

	client.initClient()
	client.registerScripts()

	return client
}

// initClient initializes the Redis client
func (c *Client) initClient() {
	c.client = redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", c.config.Host, c.config.Port),
		Password:        c.config.Password,
		DB:              c.config.DB,
		PoolSize:        c.config.MaxConnections,
		MinIdleConns:    10,
		DialTimeout:     c.config.ConnTimeout,
		ReadTimeout:     c.config.ReadTimeout,
		WriteTimeout:    c.config.WriteTimeout,
		PoolTimeout:     4 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,
	})
}

// registerScripts registers Lua scripts for atomic operations
func (c *Client) registerScripts() {
	// Script for safe lock release
	c.scripts["releaseLock"] = redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	// Script for extending lock
	c.scripts["extendLock"] = redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	// Script for scanning and deleting keys by pattern
	c.scripts["deleteByPattern"] = redis.NewScript(`
		local cursor = "0"
		local keyCount = 0
		repeat
			local result = redis.call("SCAN", cursor, "MATCH", ARGV[1], "COUNT", 100)
			cursor = result[1]
			local keys = result[2]
			if #keys > 0 then
				keyCount = keyCount + #keys
				redis.call("DEL", unpack(keys))
			end
		until cursor == "0"
		return keyCount
	`)

	// Script for bulk key operations
	c.scripts["bulkOperation"] = redis.NewScript(`
		local operation = ARGV[1]
		local result = {}

		if operation == "DEL" then
			for i = 2, #ARGV do
				local pattern = ARGV[i]
				local count = 0
				local cursor = "0"
				repeat
					local scanResult = redis.call("SCAN", cursor, "MATCH", pattern, "COUNT", 100)
					cursor = scanResult[1]
					local keys = scanResult[2]
					if #keys > 0 then
						count = count + redis.call("DEL", unpack(keys))
					end
				until cursor == "0"
				result[i-1] = count
			end
		elseif operation == "EXISTS" then
			for i = 2, #ARGV do
				result[i-1] = redis.call("EXISTS", ARGV[i])
			end
		end

		return result
	`)

	// Script for atomic cache refresh
	c.scripts["refreshCache"] = redis.NewScript(`
		local key = KEYS[1]
		local value = ARGV[1]
		local expiry = tonumber(ARGV[2])
		local refreshKey = "refresh:" .. key
		local refreshExpiry = math.floor(expiry * 0.75)

		redis.call("SET", key, value, "PX", expiry)
		redis.call("SET", refreshKey, "1", "PX", refreshExpiry)
		return 1
	`)
}

// checkAndResetClient checks if we should reset the client due to errors
func (c *Client) checkAndResetClient() {
	currentTime := time.Now().Unix()
	errorCount := atomic.LoadInt32(&c.errorCount)
	lastErrorTime := atomic.LoadInt64(&c.lastErrorTime)

	// Reset client if too many errors recently
	if errorCount > 5 && (currentTime-lastErrorTime) < 60 {
		c.mu.Lock()
		defer c.mu.Unlock()

		log.Printf("Too many Redis errors, resetting connection")
		if c.client != nil {
			_ = c.client.Close()
		}

		c.initClient()
		c.registerScripts()

		// Reset error counter
		atomic.StoreInt32(&c.errorCount, 0)
	}
}

// recordError records an error occurrence for monitoring purposes
func (c *Client) recordError() {
	currentTime := time.Now().Unix()
	atomic.StoreInt64(&c.lastErrorTime, currentTime)
	atomic.AddInt32(&c.errorCount, 1)
}

// Ping checks if Redis is responding
func (c *Client) Ping(ctx context.Context) error {
	c.checkAndResetClient()

	_, err := c.client.Ping(ctx).Result()
	if err != nil {
		c.recordError()
		return fmt.Errorf("redis ping error: %w", err)
	}

	return nil
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	c.checkAndResetClient()

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // Key doesn't exist but not an error
		}
		c.recordError()
		return "", fmt.Errorf("redis get error: %w", err)
	}

	return val, nil
}

// GetJSON retrieves and parses a JSON value
func (c *Client) GetJSON(ctx context.Context, key string, result any) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	if data == "" {
		return redis.Nil
	}

	err = json.Unmarshal([]byte(data), result)
	if err != nil {
		return fmt.Errorf("json unmarshal error: %w", err)
	}

	return nil
}

// Set sets a value with expiration
func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	c.checkAndResetClient()

	err := c.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		c.recordError()
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// SetJSON serializes and stores a JSON value
func (c *Client) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	return c.Set(ctx, key, data, expiration)
}

// Delete removes a key
func (c *Client) Delete(ctx context.Context, key string) (bool, error) {
	c.checkAndResetClient()

	result, err := c.client.Del(ctx, key).Result()
	if err != nil {
		c.recordError()
		return false, fmt.Errorf("redis delete error: %w", err)
	}

	return result > 0, nil
}

// DeleteMany deletes multiple keys
func (c *Client) DeleteMany(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	c.checkAndResetClient()

	result, err := c.client.Del(ctx, keys...).Result()
	if err != nil {
		c.recordError()
		return 0, fmt.Errorf("redis delete many error: %w", err)
	}

	return result, nil
}

// TTL gets the remaining time to live of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.checkAndResetClient()

	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		c.recordError()
		return 0, fmt.Errorf("redis ttl error: %w", err)
	}

	if ttl == -2*time.Second {
		return 0, redis.Nil // Key doesn't exist
	}

	return ttl, nil
}

// Expire sets a key's time to live in seconds
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	c.checkAndResetClient()

	result, err := c.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		c.recordError()
		return false, fmt.Errorf("redis expire error: %w", err)
	}

	return result, nil
}

// Pipeline executes multiple commands in a pipeline for better performance
func (c *Client) Pipeline(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	c.checkAndResetClient()

	pipe := c.client.Pipeline()
	err := fn(pipe)
	if err != nil {
		return nil, fmt.Errorf("pipeline function error: %w", err)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.recordError()
		return nil, fmt.Errorf("redis pipeline error: %w", err)
	}

	return cmds, nil
}

// BatchSetJSON efficiently sets multiple JSON objects in a single pipeline operation
func (c *Client) BatchSetJSON(ctx context.Context, keyValuePairs map[string]any, expiration time.Duration) error {
	if len(keyValuePairs) == 0 {
		return nil
	}

	c.checkAndResetClient()

	// Create a pipeline for batch operations
	pipe := c.client.Pipeline()

	// Add all JSON objects to the pipeline
	for key, value := range keyValuePairs {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("json marshal error for key %s: %w", key, err)
		}

		pipe.Set(ctx, key, data, expiration)
	}

	// Execute the pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		c.recordError()
		return fmt.Errorf("batch set json error: %w", err)
	}

	return nil
}

// BatchDeleteKeys deletes multiple keys in a single operation
func (c *Client) BatchDeleteKeys(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	c.checkAndResetClient()

	// For small sets of keys, use Del directly
	if len(keys) <= 100 {
		result, err := c.client.Del(ctx, keys...).Result()
		if err != nil {
			c.recordError()
			return 0, fmt.Errorf("batch delete error: %w", err)
		}
		return result, nil
	}

	// For larger sets, split into batches of 100 and use pipeline
	var totalDeleted int64

	for i := 0; i < len(keys); i += 100 {
		end := i + 100
		// if end > len(keys) {
		// 	end = len(keys)
		// }
		end = min(end, len(keys))

		batch := keys[i:end]

		// Use pipeline for this batch
		pipe := c.client.Pipeline()
		pipe.Del(ctx, batch...)

		cmds, err := pipe.Exec(ctx)
		if err != nil {
			c.recordError()
			return totalDeleted, fmt.Errorf("batch delete error on batch %d: %w", i/100, err)
		}

		// Add up deleted keys
		if len(cmds) > 0 {
			delCmd := cmds[0]
			if delResult, ok := delCmd.(*redis.IntCmd); ok {
				deleted, err := delResult.Result()
				if err == nil {
					totalDeleted += deleted
				}
			}
		}
	}

	return totalDeleted, nil
}

// MGet retrieves multiple values in a single operation
func (c *Client) MGet(ctx context.Context, keys []string) ([]any, error) {
	if len(keys) == 0 {
		return []any{}, nil
	}

	c.checkAndResetClient()

	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		c.recordError()
		return nil, fmt.Errorf("redis mget error: %w", err)
	}

	return values, nil
}

// BatchGetJSON retrieves multiple JSON values in a single operation and unmarshals them
func (c *Client) BatchGetJSON(ctx context.Context, keys []string, results map[string]any) error {
	if len(keys) == 0 || results == nil {
		return nil
	}

	// Get all values at once
	values, err := c.MGet(ctx, keys)
	if err != nil {
		return err
	}

	// Process results
	for i, value := range values {
		if value == nil {
			// Key doesn't exist
			continue
		}

		if strValue, ok := value.(string); ok {
			// Create a new instance of the appropriate type
			targetValue, ok := results[keys[i]]
			if !ok {
				continue // Skip if no target exists
			}

			// Unmarshal the JSON
			err := json.Unmarshal([]byte(strValue), targetValue)
			if err != nil {
				return fmt.Errorf("json unmarshal error for key %s: %w", keys[i], err)
			}
		}
	}

	return nil
}

// ScanKeys scans for keys matching a pattern
func (c *Client) ScanKeys(ctx context.Context, pattern string) ([]string, error) {
	c.checkAndResetClient()

	var cursor uint64 = 0
	var allKeys []string

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.recordError()
			return nil, fmt.Errorf("redis scan error: %w", err)
		}

		allKeys = append(allKeys, keys...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return allKeys, ctx.Err()
		default:
			// Continue scanning
		}
	}

	return allKeys, nil
}

// DeleteByPattern deletes all keys matching a pattern
func (c *Client) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	c.checkAndResetClient()

	// Use the Lua script for atomic pattern delete
	script := c.scripts["deleteByPattern"]
	result, err := script.Run(ctx, c.client, []string{}, pattern).Int64()
	if err != nil {
		c.recordError()
		return 0, fmt.Errorf("redis delete by pattern error: %w", err)
	}

	return result, nil
}

// BulkDeleteByPatterns deletes multiple patterns in a single operation
func (c *Client) BulkDeleteByPatterns(ctx context.Context, patterns []string) ([]int64, error) {
	if len(patterns) == 0 {
		return []int64{}, nil
	}

	c.checkAndResetClient()

	// Convert string array to []any
	args := make([]any, len(patterns)+1)
	args[0] = "DEL"
	for i, pattern := range patterns {
		args[i+1] = pattern
	}

	// Use the bulk operation script
	script := c.scripts["bulkOperation"]
	result, err := script.Run(ctx, c.client, []string{}, args...).Int64Slice()
	if err != nil {
		c.recordError()
		return nil, fmt.Errorf("redis bulk delete by patterns error: %w", err)
	}

	return result, nil
}

// AcquireLock acquires a distributed lock
func (c *Client) AcquireLock(ctx context.Context, lockName string, expiration time.Duration, retryCount int, retryDelay time.Duration) (bool, error) {
	lockID := uuid.New().String()
	lockKey := fmt.Sprintf("lock:%s", lockName)

	for attempt := 0; attempt < retryCount; attempt++ {
		c.checkAndResetClient()

		// Try to acquire lock
		acquired, err := c.client.SetNX(ctx, lockKey, lockID, expiration).Result()
		if err != nil {
			c.recordError()
			log.Printf("Error acquiring lock %s: %v", lockName, err)

			// Wait before retry
			if attempt < retryCount-1 {
				time.Sleep(retryDelay * 2)
			}
			continue
		}

		if acquired {
			// Store lock for later release
			c.mu.Lock()
			c.locks[lockKey] = lockID
			c.mu.Unlock()

			log.Printf("Lock acquired: %s with ID %s", lockKey, lockID)
			return true, nil
		}

		// If we didn't get the lock, wait before retrying
		if attempt < retryCount-1 {
			time.Sleep(retryDelay)
		}
	}

	return false, nil
}

// ReleaseLock releases a distributed lock
func (c *Client) ReleaseLock(ctx context.Context, lockName string) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", lockName)

	c.mu.Lock()
	lockID, exists := c.locks[lockKey]
	c.mu.Unlock()

	if !exists {
		log.Printf("Attempted to release lock %s that wasn't acquired by this instance", lockName)
		return false, nil
	}

	c.checkAndResetClient()

	// Use Lua script for atomic release
	script := c.scripts["releaseLock"]
	result, err := script.Run(ctx, c.client, []string{lockKey}, lockID).Int64()
	if err != nil {
		c.recordError()
		return false, fmt.Errorf("error releasing lock: %w", err)
	}

	// Clean up our local tracking
	c.mu.Lock()
	delete(c.locks, lockKey)
	c.mu.Unlock()

	return result == 1, nil
}

// ExtendLock extends the expiration time of a lock
func (c *Client) ExtendLock(ctx context.Context, lockName string, additionalTime time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", lockName)

	c.mu.Lock()
	lockID, exists := c.locks[lockKey]
	c.mu.Unlock()

	if !exists {
		log.Printf("Attempted to extend lock %s that wasn't acquired by this instance", lockName)
		return false, nil
	}

	c.checkAndResetClient()

	// Use Lua script for atomic extension
	script := c.scripts["extendLock"]
	result, err := script.Run(ctx, c.client, []string{lockKey}, lockID, int(additionalTime.Seconds())).Int64()
	if err != nil {
		c.recordError()
		return false, fmt.Errorf("error extending lock: %w", err)
	}

	if result != 1 {
		// Lock no longer exists or is owned by someone else
		c.mu.Lock()
		delete(c.locks, lockKey)
		c.mu.Unlock()
	}

	return result == 1, nil
}

// IsLocked checks if a lock exists
func (c *Client) IsLocked(ctx context.Context, lockName string) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", lockName)

	c.checkAndResetClient()

	exists, err := c.client.Exists(ctx, lockKey).Result()
	if err != nil {
		c.recordError()
		return false, fmt.Errorf("error checking lock: %w", err)
	}

	return exists == 1, nil
}

// GetWithFallback gets data from cache with callback fallback
func (c *Client) GetWithFallback(ctx context.Context, key string, fallback func(ctx context.Context) (any, error), expiration time.Duration) (any, error) {
	// Try to get from cache first
	var result any
	err := c.GetJSON(ctx, key, &result)
	if err == nil {
		return result, nil
	}

	if err != redis.Nil {
		log.Printf("Error getting data from cache: %v", err)
	}

	// Get data from fallback function
	data, err := fallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("fallback function error: %w", err)
	}

	if data == nil {
		return nil, nil
	}

	// Cache the result in a goroutine to avoid blocking
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if expiration == 0 {
			expiration = DefaultCacheExpiry * time.Second
		}

		err := c.SetJSON(cacheCtx, key, data, expiration)
		if err != nil {
			log.Printf("Failed to cache data for key %s: %v", key, err)
		}
	}()

	return data, nil
}

// SetWithRefresh sets a value with both main expiration and refresh marker
func (c *Client) SetWithRefresh(ctx context.Context, key string, value any, expiration time.Duration) error {
	c.checkAndResetClient()

	// Marshal the value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	// Use atomic refresh script
	script := c.scripts["refreshCache"]
	_, err = script.Run(ctx, c.client, []string{key}, string(data), int(expiration.Milliseconds())).Result()
	if err != nil {
		c.recordError()
		return fmt.Errorf("redis set with refresh error: %w", err)
	}

	return nil
}

// SAdd adds members to a set
func (c *Client) SAdd(ctx context.Context, key string, members ...any) (int64, error) {
	c.checkAndResetClient()

	result, err := c.client.SAdd(ctx, key, members...).Result()
	if err != nil {
		c.recordError()
		return 0, fmt.Errorf("redis sadd error: %w", err)
	}

	return result, nil
}

// SRem removes members from a set
func (c *Client) SRem(ctx context.Context, key string, members ...any) (int64, error) {
	c.checkAndResetClient()

	result, err := c.client.SRem(ctx, key, members...).Result()
	if err != nil {
		c.recordError()
		return 0, fmt.Errorf("redis srem error: %w", err)
	}

	return result, nil
}

// SMembers gets all members of a set
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	c.checkAndResetClient()

	result, err := c.client.SMembers(ctx, key).Result()
	if err != nil {
		c.recordError()
		return nil, fmt.Errorf("redis smembers error: %w", err)
	}

	return result, nil
}

// SIsMember checks if a value is a member of a set
func (c *Client) SIsMember(ctx context.Context, key string, member any) (bool, error) {
	c.checkAndResetClient()

	result, err := c.client.SIsMember(ctx, key, member).Result()
	if err != nil {
		c.recordError()
		return false, fmt.Errorf("redis sismember error: %w", err)
	}

	return result, nil
}

// SCard gets the number of members in a set
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	c.checkAndResetClient()

	result, err := c.client.SCard(ctx, key).Result()
	if err != nil {
		c.recordError()
		return 0, fmt.Errorf("redis scard error: %w", err)
	}

	return result, nil
}

// Eval executes a Lua script in Redis
func (c *Client) Eval(ctx context.Context, script string, keys []string, args []string) (any, error) {
	c.checkAndResetClient()

	// Create a script object
	luaScript := redis.NewScript(script)

	// Execute the script with the provided keys and args
	result, err := luaScript.Run(ctx, c.client, keys, args).Result()
	if err != nil && err != redis.Nil {
		c.recordError()
		return nil, fmt.Errorf("redis eval error: %w", err)
	}

	return result, nil
}

// EvalSha executes a script cached on the server side by its SHA1 digest
func (c *Client) EvalSha(ctx context.Context, sha1 string, keys []string, args []string) (any, error) {
	c.checkAndResetClient()

	result, err := c.client.EvalSha(ctx, sha1, keys, args).Result()
	if err != nil && err != redis.Nil {
		c.recordError()
		return nil, fmt.Errorf("redis evalsha error: %w", err)
	}

	return result, nil
}

// ScriptLoad loads a script into the script cache, but does not execute it
func (c *Client) ScriptLoad(ctx context.Context, script string) (string, error) {
	c.checkAndResetClient()

	sha1, err := c.client.ScriptLoad(ctx, script).Result()
	if err != nil {
		c.recordError()
		return "", fmt.Errorf("redis script load error: %w", err)
	}

	return sha1, nil
}

// ScriptExists returns information about the existence of the scripts in the script cache
func (c *Client) ScriptExists(ctx context.Context, scripts ...string) ([]bool, error) {
	c.checkAndResetClient()

	result, err := c.client.ScriptExists(ctx, scripts...).Result()
	if err != nil {
		c.recordError()
		return nil, fmt.Errorf("redis script exists error: %w", err)
	}

	return result, nil
}

// ScriptFlush flushes the script cache
func (c *Client) ScriptFlush(ctx context.Context) error {
	c.checkAndResetClient()

	err := c.client.ScriptFlush(ctx).Err()
	if err != nil {
		c.recordError()
		return fmt.Errorf("redis script flush error: %w", err)
	}

	return nil
}

// Close closes the Redis client
func (c *Client) Close() error {
	return c.client.Close()
}
