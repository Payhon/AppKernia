import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const docsRoot = path.resolve(fileURLToPath(new URL('..', import.meta.url)), 'docs');
const repositoryRoot = path.resolve(docsRoot, '../../..');
const openapiPath = path.join(repositoryRoot, 'server/openapi/openapi.yaml');
const openapi = await readFile(openapiPath, 'utf8');
const openapiPaths = new Set(
  [...openapi.matchAll(/^ {2}(\/[^:\n]+):\s*$/gm)].map((match) => match[1]),
);
const failures = [];
let referenceCount = 0;

if (!/^\s*license:\s*\n\s+name:\s+MIT\s*$/m.test(openapi)) {
  failures.push('OpenAPI info.license must declare MIT.');
}

for (const locale of ['zh-CN', 'en-US']) {
  const apiDirectory = path.join(docsRoot, locale, 'api');
  const files = (await readdir(apiDirectory)).filter(
    (file) => file.endsWith('.md') && !['index.md', 'conventions.md'].includes(file),
  );

  for (const file of files) {
    const prefix = file.startsWith('mobile-') ? '/api/v1' : '/admin-api/v1';
    const content = await readFile(path.join(apiDirectory, file), 'utf8');
    const references = [...content.matchAll(/`(\/[^`\s]+)`/g)].map((match) => match[1]);

    for (const reference of references) {
      if (reference === '/api/v1' || reference === '/admin-api/v1') continue;
      referenceCount += 1;
      if (reference.includes('|')) {
        failures.push(`${locale}/api/${file}: compound path ${reference} must be expanded.`);
        continue;
      }

      const withoutQuery = reference.split('?')[0];
      const fullPath = withoutQuery.startsWith('/internal/')
        ? withoutQuery
        : `${prefix}${withoutQuery}`;
      if (!openapiPaths.has(fullPath)) {
        failures.push(`${locale}/api/${file}: ${fullPath} is absent from OpenAPI.`);
      }
    }
  }
}

if (failures.length > 0) {
  console.error(failures.join('\n'));
  process.exit(1);
}

console.log(`Validated ${referenceCount} documented API path references against OpenAPI.`);
