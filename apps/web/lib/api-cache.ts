type CacheEntry<T> = {
  data?: T;
  error?: string;
  promise?: Promise<T>;
  updatedAt: number;
};

const apiCache = new Map<string, CacheEntry<unknown>>();

export function getApiCache<T>(path: string) {
  return apiCache.get(path) as CacheEntry<T> | undefined;
}

export function setApiCache<T>(path: string, entry: CacheEntry<T>) {
  apiCache.set(path, entry as CacheEntry<unknown>);
}

export function invalidateApiCache(match?: string | RegExp) {
  if (!match) {
    apiCache.clear();
    return;
  }
  for (const key of apiCache.keys()) {
    if (typeof match === "string" ? key.includes(match) : match.test(key)) apiCache.delete(key);
  }
}
