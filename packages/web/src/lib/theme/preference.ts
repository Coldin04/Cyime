import { browser } from '$app/environment';

export type ThemeMode = 'system' | 'light' | 'dark';

export const THEME_MODE_STORAGE_KEY = 'cyime-theme-mode';

const VALID_THEME_MODES: ThemeMode[] = ['system', 'light', 'dark'];

function isThemeMode(value: string | null | undefined): value is ThemeMode {
	return !!value && VALID_THEME_MODES.includes(value as ThemeMode);
}

export function getStoredThemeMode(): ThemeMode {
	if (!browser) return 'system';
	const stored = localStorage.getItem(THEME_MODE_STORAGE_KEY);
	return isThemeMode(stored) ? stored : 'system';
}

export function setStoredThemeMode(mode: ThemeMode): void {
	if (!browser) return;
	localStorage.setItem(THEME_MODE_STORAGE_KEY, mode);
}

function prefersSystemDark(): boolean {
	if (!browser) return false;
	return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

export function applyThemeMode(mode: ThemeMode): void {
	if (!browser) return;
	const shouldUseDark = mode === 'dark' || (mode === 'system' && prefersSystemDark());
	document.documentElement.classList.toggle('dark', shouldUseDark);
	document.documentElement.style.colorScheme = shouldUseDark ? 'dark' : 'light';
}

export function initThemeModeSync(): () => void {
	if (!browser) return () => {};

	const media = window.matchMedia('(prefers-color-scheme: dark)');
	const sync = () => applyThemeMode(getStoredThemeMode());
	sync();

	const onChange = () => {
		if (getStoredThemeMode() === 'system') {
			applyThemeMode('system');
		}
	};

	if (typeof media.addEventListener === 'function') {
		media.addEventListener('change', onChange);
		return () => media.removeEventListener('change', onChange);
	}

	media.addListener(onChange);
	return () => media.removeListener(onChange);
}
