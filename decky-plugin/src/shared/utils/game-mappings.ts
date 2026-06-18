const STORAGE_KEY = 'sentinel-game-mappings';

export interface GameMapping {
  sentinelAppId: string;
  sentinelName: string;
  shortcutName: string;
  createdAt: number;
}

function readMappings(): Record<number, GameMapping> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

function writeMappings(mappings: Record<number, GameMapping>): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(mappings));
  } catch {
    // localStorage unavailable
  }
}

export function getMapping(nonSteamAppId: number): string | null {
  return readMappings()[nonSteamAppId]?.sentinelAppId ?? null;
}

export function setMapping(
  nonSteamAppId: number,
  sentinelAppId: string,
  sentinelName: string,
  shortcutName: string
): void {
  const mappings = readMappings();
  mappings[nonSteamAppId] = {
    sentinelAppId,
    sentinelName,
    shortcutName,
    createdAt: Date.now()
  };
  writeMappings(mappings);
}

export function getAllMappings(): Record<number, GameMapping> {
  return readMappings();
}

export function clearMapping(nonSteamAppId: number): void {
  const mappings = readMappings();
  delete mappings[nonSteamAppId];
  writeMappings(mappings);
}
