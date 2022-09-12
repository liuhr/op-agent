package util

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

var logEntryCache *cache.Cache = cache.New(time.Minute, time.Second*5)

func ClearToLog(topic string, key string) bool {
	return logEntryCache.Add(fmt.Sprintf("%s:%s", topic, key), true, cache.DefaultExpiration) == nil
}
