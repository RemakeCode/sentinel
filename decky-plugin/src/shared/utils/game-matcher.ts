import type { GameBasics } from '@/shared/types/GameBasics';

function normalize(s: string): string {
  return s.toLowerCase().trim();
}

function wordOverlap(a: string, b: string): number {
  const wordsA = new Set(a.split(/\s+/));
  const wordsB = new Set(b.split(/\s+/));
  let intersection = 0;
  for (const w of wordsA) {
    if (wordsB.has(w)) intersection++;
  }
  const union = wordsA.size + wordsB.size - intersection;
  return union === 0 ? 0 : intersection / union;
}

export function matchGameByName(name: string, games: GameBasics[]): GameBasics | null {
  const normalized = normalize(name);

  for (const game of games) {
    if (normalize(game.Name) === normalized) return game;
  }

  for (const game of games) {
    const gName = normalize(game.Name);
    if (gName.includes(normalized) || normalized.includes(gName)) return game;
  }

  let best: GameBasics | null = null;
  let bestScore = 0;
  for (const game of games) {
    const score = wordOverlap(normalized, normalize(game.Name));
    if (score > bestScore) {
      bestScore = score;
      best = game;
    }
  }

  return bestScore >= 0.3 ? best : null;
}
