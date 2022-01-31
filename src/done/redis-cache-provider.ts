import { createClient } from 'redis';
import { REDIS_URL } from './environment_variables.js';

async function RedisCache(): Promise<CPCCache> {
    const client = createClient({
        url: REDIS_URL
    });

    client.on('error', (err) => console.log('Redis Client Error', err));

    await client.connect();

    function get_redisKey(namespace: string, key: string | number) {
        return `${namespace}:->'${key}`;
    }

    async function cacheCheck(namespace: string, key: string | number): Promise<boolean> {
        const redis_key = get_redisKey(namespace, key);
        return client.exists(redis_key);
    }

    async function cacheGet(namespace: string, key: string | number): Promise<any> {
        let hit = undefined;
        const redis_key = get_redisKey(namespace, key);
        hit = await client.get(redis_key);
        if (hit === undefined || hit === null) {
            throw new Error('undefined cache is in error');
        }
        return JSON.parse(hit);
    }

    async function cacheSet(namespace: string, key: string | number, data: any, expiration_period?: number | undefined): Promise<void> {
        if (data === undefined) {
            throw new Error('Cannot cache undefined');
        }
        const redis_key = get_redisKey(namespace, key);
        if (expiration_period !== undefined) {
            client.setEx(redis_key, expiration_period, JSON.stringify(data));
        } else {
            client.set(redis_key, JSON.stringify(data));
        }
    }

    async function shutdown() {
        client.disconnect();
    }

    return Object.freeze({
        cacheCheck,
        cacheGet,
        cacheSet,
        shutdown
    })
}

export {
    RedisCache
}