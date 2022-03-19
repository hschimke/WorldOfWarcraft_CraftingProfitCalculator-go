package cache_provider

import (
	"encoding/json"
	"fmt"
	"time"
)

// Create a Redis key given a namespace and key
func getRedisKey(namespace string, key string) string {
	return fmt.Sprint(namespace, ":->", key)
}

// Fetch an object from the cache, or fail
func CacheGet[T any](cache *CacheProvider, namespace string, key string, target *T) error {
	data, getErr := cache.redisClient.Get(cache.ctx, getRedisKey(namespace, key)).Result()
	if getErr != nil {
		return getErr
	}
	//convertErr := gob.NewDecoder(bytes.NewReader([]byte(data))).Decode(&target)
	//_, convertErr := asn1.Unmarshal([]byte(data), &target)
	convertErr := json.Unmarshal([]byte(data), &target)
	if convertErr != nil {
		return convertErr
	}
	return nil
}

// Set or replace an item in the cache, or fail.
func CacheSet[T any](cache *CacheProvider, namespace string, key string, data T, expiration_period time.Duration) error {
	//var buf bytes.Buffer
	//bufW := bufio.NewWriter(&buf)
	//err := gob.NewEncoder(bufW).Encode(&data)
	//asn1_data, err := asn1.Marshal(&data)
	json_data, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	//bufW.Flush()
	setErr := cache.redisClient.Set(cache.ctx, getRedisKey(namespace, key), json_data, expiration_period).Err()
	//setErr := cache.redisClient.Set(cache.ctx, getRedisKey(namespace, key), asn1_data, expiration_period).Err()
	//setErr := cache.redisClient.Set(cache.ctx, getRedisKey(namespace, key), buf.Bytes(), expiration_period).Err()
	if setErr != nil {
		return setErr
	}
	return nil
}

// Check if a given key exists in a given namespace
func CacheCheck(cache *CacheProvider, namespace string, key string) (bool, error) {
	//return false, nil
	fnd, err := cache.redisClient.Exists(cache.ctx, getRedisKey(namespace, key)).Result()
	if err != nil {
		return false, err
	}
	return fnd == 1, nil
}
