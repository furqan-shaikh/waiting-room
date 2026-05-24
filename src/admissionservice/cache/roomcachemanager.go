package cache

import (
	"fmt"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type RoomCacheManagerConfig struct {
	TTLInMinutes time.Duration
}

type RoomCacheManager struct {
	roomCache *ttlcache.Cache[string, string]
}

func NewRoomCacheManager(config RoomCacheManagerConfig) *RoomCacheManager {
	c := ttlcache.New(
		ttlcache.WithTTL[string, string](config.TTLInMinutes),
	)

	go c.Start()

	return &RoomCacheManager{
		roomCache: c,
	}
}

func (cacheManager *RoomCacheManager) Get(key string) (string, error) {
	if item := cacheManager.roomCache.Get(key); item != nil {
		return item.Value(), nil
	}
	return "", fmt.Errorf("cache miss for %v", key)
}

func (cacheManager *RoomCacheManager) Set(key string, value string) {
	cacheManager.roomCache.Set(key, value, ttlcache.DefaultTTL)
}

func (cacheManager *RoomCacheManager) Purge() {
	cacheManager.roomCache.DeleteAll()
}
