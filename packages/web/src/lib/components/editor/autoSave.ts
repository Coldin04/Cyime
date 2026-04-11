const AUTO_SAVE_ENABLED_KEY = 'cyimewrite.editor.autoSave.enabled';
const AUTO_SAVE_INTERVAL_KEY = 'cyimewrite.editor.autoSave.intervalSeconds';
const envDefaultAutoSaveIntervalSeconds = Number.parseInt(
	import.meta.env.PUBLIC_EDITOR_AUTO_SAVE_INTERVAL_SECONDS || '10',
	10
);

export const defaultAutoSaveEnabled = true;
export const minAutoSaveIntervalSeconds = 1;
export const maxAutoSaveIntervalSeconds = 300;

function clampAutoSaveIntervalValue(value: number): number {
	return Math.min(maxAutoSaveIntervalSeconds, Math.max(minAutoSaveIntervalSeconds, Math.round(value)));
}

export const defaultAutoSaveIntervalSeconds = Number.isFinite(envDefaultAutoSaveIntervalSeconds)
	? clampAutoSaveIntervalValue(envDefaultAutoSaveIntervalSeconds)
	: 10;

export function clampAutoSaveInterval(value: number): number {
	if (!Number.isFinite(value)) {
		return defaultAutoSaveIntervalSeconds;
	}

	return clampAutoSaveIntervalValue(value);
}

export function readAutoSaveEnabled(): boolean {
	if (typeof localStorage === 'undefined') {
		return defaultAutoSaveEnabled;
	}

	const stored = localStorage.getItem(AUTO_SAVE_ENABLED_KEY);
	if (stored === null) {
		return defaultAutoSaveEnabled;
	}

	return stored === 'true';
}

export function writeAutoSaveEnabled(enabled: boolean) {
	if (typeof localStorage === 'undefined') {
		return;
	}

	localStorage.setItem(AUTO_SAVE_ENABLED_KEY, String(enabled));
}

export function readAutoSaveIntervalSeconds(): number {
	if (typeof localStorage === 'undefined') {
		return defaultAutoSaveIntervalSeconds;
	}

	const stored = localStorage.getItem(AUTO_SAVE_INTERVAL_KEY);
	if (!stored) {
		return defaultAutoSaveIntervalSeconds;
	}

	return clampAutoSaveInterval(Number.parseInt(stored, 10));
}

export function writeAutoSaveIntervalSeconds(seconds: number) {
	if (typeof localStorage === 'undefined') {
		return;
	}

	localStorage.setItem(AUTO_SAVE_INTERVAL_KEY, String(clampAutoSaveInterval(seconds)));
}
