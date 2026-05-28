const STORAGE_KEY = 'sentinel-game-mappings';

export function getMapping(nonSteamAppId: number): string | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const mappings: Record<number, string> = JSON.parse(raw);
    return mappings[nonSteamAppId] ?? null;
  } catch {
    return null;
  }
}

export function setMapping(nonSteamAppId: number, sentinelAppId: string): void {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const mappings: Record<number, string> = raw ? JSON.parse(raw) : {};
    mappings[nonSteamAppId] = sentinelAppId;
    localStorage.setItem(STORAGE_KEY, JSON.stringify(mappings));
  } catch {
    // localStorage unavailable
  }
}
