import { execFileSync } from 'node:child_process';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

export async function prepare(_, { cwd, env, nextRelease }) {
  const directory = await mkdtemp(join(tmpdir(), 'sentinel-release-'));
  try {
    const notes = join(directory, 'notes.md');
    await writeFile(notes, nextRelease.notes ?? '');
    const repository = env.GITHUB_REPOSITORY || 'RemakeCode/sentinel';
    const prefix = `https://github.com/${repository}/releases/download/${nextRelease.gitTag}`;

    for (const [archive, manifest] of [
      ['sentinel-linux.tar.gz', 'update.json'],
      ['sentinel-flatpak-linux.tar.gz', 'update-flatpak.json'],
    ]) {
      execFileSync('wails3', [
        'updater', 'manifest',
        '-version', nextRelease.version,
        '-channel', 'stable',
        '-name', nextRelease.gitTag,
        '-notes-file', notes,
        '-url-prefix', prefix,
        '-output', join('bin', manifest),
        join('bin', archive),
      ], { cwd, env, stdio: 'inherit' });
    }
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
}
