import { copyFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const projectDirectory = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repositoryDirectory = path.resolve(projectDirectory, '../..');
const source = path.join(repositoryDirectory, 'server/openapi/openapi.yaml');
const destination = path.join(projectDirectory, 'docs/public/openapi.yaml');

await mkdir(path.dirname(destination), { recursive: true });
await copyFile(source, destination);

process.stdout.write(
  `Synced ${path.relative(repositoryDirectory, source)} -> ${path.relative(repositoryDirectory, destination)}\n`,
);
