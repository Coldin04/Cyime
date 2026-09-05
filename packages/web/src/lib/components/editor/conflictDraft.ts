import type { JSONContent } from '@tiptap/core';

const STORAGE_PREFIX = 'cyime.editor.conflictDraft.';

export type ConflictDraft = {
	/** contentVersion the editor had loaded when this draft was captured. */
	baseContentVersion: number;
	baseContentJson: JSONContent;
	localContentJson: JSONContent;
	savedAt: string;
};

function storageKey(documentId: string): string {
	return `${STORAGE_PREFIX}${documentId}`;
}

/**
 * Persists the in-progress local draft the moment a save conflict is
 * detected, so an accidental refresh/tab-close while the resolution dialog
 * is still open does not silently discard unsaved edits.
 */
export function saveConflictDraft(documentId: string, draft: ConflictDraft): void {
	try {
		window.localStorage.setItem(storageKey(documentId), JSON.stringify(draft));
	} catch (error) {
		console.error('[ConflictDraft] Failed to persist local draft:', error);
	}
}

export function loadConflictDraft(documentId: string): ConflictDraft | null {
	try {
		const raw = window.localStorage.getItem(storageKey(documentId));
		if (!raw) return null;
		const parsed = JSON.parse(raw) as Partial<ConflictDraft>;
		if (
			!parsed ||
			typeof parsed.baseContentVersion !== 'number' ||
			!parsed.localContentJson ||
			!parsed.baseContentJson
		) {
			return null;
		}
		return parsed as ConflictDraft;
	} catch (error) {
		console.error('[ConflictDraft] Failed to read local draft:', error);
		return null;
	}
}

export function clearConflictDraft(documentId: string): void {
	try {
		window.localStorage.removeItem(storageKey(documentId));
	} catch (error) {
		console.error('[ConflictDraft] Failed to clear local draft:', error);
	}
}
