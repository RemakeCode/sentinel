import fs from 'node:fs';
import path from 'node:path';

const manifestPath = path.resolve(import.meta.dirname, '../out/sentinel/plugin.json');
const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
const flags = Array.isArray(manifest.flags) ? manifest.flags.filter((flag) => flag && flag !== 'debug') : [];

if (process.env.SENTINEL_DECKY_DEBUG === '1') flags.push('debug');
manifest.flags = flags;

fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
