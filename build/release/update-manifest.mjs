import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { createReadStream } from 'node:fs';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

export async function writeDeckyManifest({ cwd, env, nextRelease }) {
  const filename = `sentinel-decky-plugin-${nextRelease.version}.zip`;
  const hash = createHash('sha256');
  let size = 0;
  for await (const chunk of createReadStream(join(cwd, 'bin', filename))) {
    hash.update(chunk);
    size += chunk.length;
  }
  const repository = env.GITHUB_REPOSITORY || 'RemakeCode/sentinel';
  const arch = env.GOARCH || env.ARCH || (process.arch === 'x64' ? 'amd64' : process.arch);
  if (!['amd64', 'arm64'].includes(arch)) throw new Error(`Unsupported Decky architecture: ${arch}`);
  const manifest = {
    schemaVersion: 1,
    version: nextRelease.version,
    channel: 'stable',
    name: nextRelease.gitTag,
    notes: nextRelease.notes ?? '',
    artifacts: [{
      url: `https://github.com/${repository}/releases/download/${nextRelease.gitTag}/${filename}`,
      filename, filetype: 'zip', platform: 'linux', arch, size,
      digestAlgo: 'sha256', digest: hash.digest('base64'),
    }],
  };
  await writeFile(join(cwd, 'bin', 'update-decky.json'), JSON.stringify(manifest, null, 2) + '\n');
  return manifest;
}

export async function prepare(_, { cwd, env, nextRelease }) {
  await writeDeckyManifest({ cwd, env, nextRelease });
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
