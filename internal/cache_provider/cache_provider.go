package cache_provider

import (
	"fmt"
	"time"
)

func CacheGet(namespace string, key string, target interface{}) error {
	return fmt.Errorf("not implemented")
}
func CacheSet(namespace string, key string, data interface{}, expiration_period time.Duration) error {
	return fmt.Errorf("not implemented")
}
func CacheCheck(namespace string, key string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
