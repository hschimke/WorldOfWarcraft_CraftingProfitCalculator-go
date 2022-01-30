import { promises as fs } from 'fs';
import { resolve } from 'path';

const cache: Map<string, Map<string | number, { cached: number, data: any }>> = new Map();

/*async function writeOut() {
    const output: Record<string, Record<string, any>> = {};
    for (const key of cache.keys()) {
        if (key !== 'fetched_auctions_data') {
            output[key] = {};
            const ns = cache.get(key);
            if (ns !== undefined) {
                for (const ns_key of ns.keys()) {
                    output[key][ns_key] = ns.get(ns_key);
                }
            }
        }
    }
    await fs.writeFile('./cache.json', JSON.stringify(output), { encoding: 'utf8' });
}*/

async function readCache(cache: Map<string, Map<string | number, { cached: number, data: any }>>): Promise<void> {
    try {
        const fd: string = await fs.readFile(resolve('./cache', 'cache.json'), { encoding: 'utf8' });
        const obj: Record<string, Record<string, any>> = JSON.parse(fd);
        for (const ns of Object.keys(obj)) {
            const lm = new Map<string, any>();
            for (const key of Object.keys(obj[ns])) {
                lm.set(key, obj[ns][key]);
            }
            cache.set(ns, lm);
        }
    } catch { }
}

async function MemoryCache(): Promise<CPCCache> {
    //const cache: Map<string, Map<string | number, { cached: number, data: any }>> = new Map();
    readCache(cache);

    async function cacheCheck(namespace: string, key: string | number, expiration_period?: number | undefined): Promise<boolean> {
        let found = false;
        if (!cache.has(namespace)) {
            found = false;
        } else {
            const ns_map = cache.get(namespace);
            if (ns_map !== undefined) {
                if (!ns_map.has(key)) {
                    found = false;
                } else {
                    const hit = ns_map.get(key);
                    if (hit === undefined) {
                        found = false;
                    } else {
                        if (expiration_period !== undefined) {
                            const now = Date.now();
                            if ((hit.cached + expiration_period) > now) {
                                found = true;
                            } else {
                                found = false;
                            }
                        } else {
                            found = true;
                        }
                    }
                }
            }
        }
        return found;
    }

    async function cacheGet(namespace: string, key: string | number): Promise<any> {
        const ns = cache.get(namespace);
        let hit = undefined;
        if (ns !== undefined) {
            if (ns.has(key)) {
                const ns_key = ns.get(key);
                if (ns_key !== undefined) {
                    hit = ns_key.data;
                }
            }
        }
        if (hit === undefined) {
            throw new Error('undefined cache is in error');
        }
        return hit;
    }

    async function cacheSet(namespace: string, key: string | number, data: any): Promise<void> {
        if (data === undefined) {
            throw new Error('Cannot cache undefined');
        }
        if (!cache.has(namespace)) {
            cache.set(namespace, new Map<string | number, { cached: number, data: any }>());
        }
        const ns = cache.get(namespace);
        if (ns !== undefined) {
            ns.set(key, {
                cached: Date.now(),
                data: data
            });
        }
    }

    async function shutdown() {
        return;
    }

    return Object.freeze({
        cacheCheck,
        cacheGet,
        cacheSet,
        shutdown
    })
}

export {
    MemoryCache
}