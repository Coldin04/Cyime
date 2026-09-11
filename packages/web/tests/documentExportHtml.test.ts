import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDirectory = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(testDirectory, '..');

test('keeps editor and standalone HTML export on the canonical document stylesheet', async () => {
	const [appCss, editorSource, exportSource, documentCss] = await Promise.all([
		readFile(path.join(webRoot, 'src/app.css'), 'utf8'),
		readFile(path.join(webRoot, 'src/lib/components/editor/Editor.svelte'), 'utf8'),
		readFile(path.join(webRoot, 'src/lib/export/documentExport.ts'), 'utf8'),
		readFile(path.join(webRoot, 'src/lib/document/document-content.css'), 'utf8')
	]);

	assert.match(appCss, /@import '\.\/lib\/document\/document-content\.css';/);
	assert.match(editorSource, /'cy-document'/);
	assert.match(exportSource, /document-content\.css\?raw/);
	assert.match(exportSource, /<html lang="en"\$\{htmlClass\}>/);
	assert.match(exportSource, /<main class="cy-document">/);
	assert.match(documentCss, /\.cy-document table \{/);
	assert.match(documentCss, /\.cy-document pre \{/);
	assert.match(documentCss, /background: var\(--cy-document-surface-muted\);/);
	assert.match(documentCss, /height: 3\.7rem;/);
	assert.doesNotMatch(exportSource, /pre\[data-language\]/);
});
