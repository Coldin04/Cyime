const EDIT_TAB_STORAGE_KEY = 'cyime.editor.tab-id';
const EDIT_LEASE_STORAGE_PREFIX = 'cyime.editor.lease.';

function createTabId(): string {
	if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
		return crypto.randomUUID();
	}

	return `tab-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

/**
 * Identifies this browser tab/window for edit-lease purposes. Backed by
 * sessionStorage rather than localStorage: localStorage is shared by every
 * tab of the same browser, which would let multiple tabs silently share one
 * editing lock. sessionStorage is unique per tab, so each tab is its own
 * "device" as far as the lease is concerned — opening a second tab for the
 * same document must explicitly take over editing — while reloading the
 * *same* tab keeps the same id and can still reclaim its own lease.
 *
 * Known limitation: some browsers copy sessionStorage when a tab is
 * duplicated or restored after being closed, which can make the copy look
 * like the original tab. That's an accepted edge case, not a security
 * boundary — the goal here is to stop accidental concurrent edits, not to
 * authenticate anything.
 */
export function getEditTabId(): string {
	const existing = window.sessionStorage.getItem(EDIT_TAB_STORAGE_KEY)?.trim();
	if (existing && existing.length >= 8) {
		return existing;
	}

	const tabId = createTabId();
	window.sessionStorage.setItem(EDIT_TAB_STORAGE_KEY, tabId);
	return tabId;
}

export function getStoredEditLeaseToken(documentId: string): string {
	return window.sessionStorage.getItem(`${EDIT_LEASE_STORAGE_PREFIX}${documentId}`)?.trim() ?? '';
}

export function storeEditLeaseToken(documentId: string, token: string): void {
	window.sessionStorage.setItem(`${EDIT_LEASE_STORAGE_PREFIX}${documentId}`, token);
}

export function clearStoredEditLeaseToken(documentId: string, token: string): void {
	const key = `${EDIT_LEASE_STORAGE_PREFIX}${documentId}`;
	if (window.sessionStorage.getItem(key) === token) {
		window.sessionStorage.removeItem(key);
	}
}
