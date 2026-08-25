import { rm, mkdir } from 'node:fs/promises';

await rm('./dist', { recursive: true, force: true });

const result = await Bun.build({
  entrypoints: ['./src/main.js'],
  outdir: './dist/src',
  target: 'browser',
  format: 'esm',
  minify: true,
});

if (!result.success) {
  throw new AggregateError(result.logs, 'Build failed');
}

await mkdir('./dist/src', { recursive: true });
await Bun.write('./dist/src/styles.css', Bun.file('./src/styles.css'));
await Bun.write('./dist/index.html', Bun.file('./index.html'));
