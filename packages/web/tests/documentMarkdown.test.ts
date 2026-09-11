import assert from 'node:assert/strict';
import test from 'node:test';
import type { JSONContent } from '@tiptap/core';
import { exportMarkdown } from '../src/lib/export/documentMarkdown.ts';

test('exports a Tiptap table as GFM markdown', () => {
	const document: JSONContent = {
		type: 'doc',
		content: [
			{
				type: 'table',
				content: [
					{
						type: 'tableRow',
						content: [
							{
								type: 'tableHeader',
								content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Name' }] }]
							},
							{
								type: 'tableHeader',
								content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Value' }] }]
							}
						]
					},
					{
						type: 'tableRow',
						content: [
							{
								type: 'tableCell',
								content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Alpha' }] }]
							},
							{
								type: 'tableCell',
								content: [{ type: 'paragraph', content: [{ type: 'text', text: '42' }] }]
							}
						]
					}
				]
			}
		]
	};

	assert.equal(
		exportMarkdown(document),
		'| Name | Value |\n| --- | --- |\n| Alpha | 42 |'
	);
});

test('escapes table delimiters and preserves line breaks inside cells', () => {
	const document: JSONContent = {
		type: 'doc',
		content: [
			{
				type: 'table',
				content: [
					{
						type: 'tableRow',
						content: [
							{
								type: 'tableHeader',
								content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Notes' }] }]
							}
						]
					},
					{
						type: 'tableRow',
						content: [
							{
								type: 'tableCell',
								content: [
									{
										type: 'paragraph',
										content: [
											{ type: 'text', text: 'A | B' },
											{ type: 'hardBreak' },
											{ type: 'text', text: 'C' }
										]
									}
								]
							}
						]
					}
				]
			}
		]
	};

	assert.equal(
		exportMarkdown(document),
		'| Notes |\n| --- |\n| A \\| B<br>C |'
	);
});

test('keeps all rows when a table has no header row', () => {
	const document: JSONContent = {
		type: 'doc',
		content: [
			{
				type: 'table',
				content: [
					{
						type: 'tableRow',
						content: [
							{
								type: 'tableCell',
								content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Alpha' }] }]
							}
						]
					},
					{
						type: 'tableRow',
						content: [
							{
								type: 'tableCell',
								content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Beta' }] }]
							}
						]
					}
				]
			}
		]
	};

	assert.equal(exportMarkdown(document), '|  |\n| --- |\n| Alpha |\n| Beta |');
});
