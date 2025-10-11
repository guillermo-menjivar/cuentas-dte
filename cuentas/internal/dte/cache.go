// internal/services/dte/cache.go
package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// CachedCredentials holds company credentials for signing
type CachedCredentials struct {
	CompanyID       uuid.UUID `json:"company_id"`
	NIT             string    `json:"nit"`
	Password        string    `json:"password"`
	Nombre          string    `json:"nombre"`
	NombreComercial *string   `json:"nombre_comercial"`
	CachedAt        time.Time `json:"cached_at"`
}

// CredentialCache provides two-tier caching (memory + Redis)
type CredentialCache struct {
	memory    *sync.Map
	redis     *redis.Client
	memoryTTL time.Duration // 5 minutes
	redisTTL  time.Duration // 1 hour
}

// NewCredentialCache creates a new two-tier credential cache
func NewCredentialCache(redisClient *redis.Client) *CredentialCache {
	return &CredentialCache{
		memory:    &sync.Map{},
		redis:     redisClient,
		memoryTTL: 5 * time.Minute,
		redisTTL:  1 * time.Hour,
	}
}

// Get retrieves credentials from cache (memory first, then Redis)
func (c *CredentialCache) Get(ctx context.Context, companyID uuid.UUID) (*CachedCredentials, bool) {
	// Check in-memory cache first (L1)
	if val, ok := c.memory.Load(companyID.String()); ok {
		creds := val.(*CachedCredentials)

		// Check if expired
		if time.Since(creds.CachedAt) < c.memoryTTL {
			return creds, true
		}

		// Expired, remove from memory
		c.memory.Delete(companyID.String())
	}

	// Check Redis cache (L2)
	key := fmt.Sprintf("dte:creds:%s", companyID.String())
	val, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		var creds CachedCredentials
		if err := json.Unmarshal([]byte(val), &creds); err == nil {
			// Store in memory cache for next time
			c.memory.Store(companyID.String(), &creds)
			return &creds, true
		}
	}

	return nil, false
}

// Set stores credentials in both cache tiers
func (c *CredentialCache) Set(ctx context.Context, creds *CachedCredentials) error {
	creds.CachedAt = time.Now()

	// Store in memory cache (L1)
	c.memory.Store(creds.CompanyID.String(), creds)

	// Store in Redis cache (L2)
	key := fmt.Sprintf("dte:creds:%s", creds.CompanyID.String())
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := c.redis.Set(ctx, key, data, c.redisTTL).Err(); err != nil {
		return fmt.Errorf("failed to cache credentials in Redis: %w", err)
	}

	return nil
}

// Invalidate removes credentials from both cache tiers
func (c *CredentialCache) Invalidate(ctx context.Context, companyID uuid.UUID) error {
	// Remove from memory
	c.memory.Delete(companyID.String())

	// Remove from Redis
	key := fmt.Sprintf("dte:creds:%s", companyID.String())
	if err := c.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate Redis cache: %w", err)
	}

	return nil
}
