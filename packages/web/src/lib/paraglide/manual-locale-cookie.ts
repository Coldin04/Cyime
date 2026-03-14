export const MANUAL_LOCALE_COOKIE_NAME = 'cyime-locale';

export function getCookieValue(rawCookie: string | null | undefined, key: string): string | undefined {
	if (!rawCookie) return undefined;
	const entry = rawCookie
		.split(';')
		.map((part) => part.trim())
		.find((part) => part.startsWith(`${key}=`));
	if (!entry) return undefined;
	const value = entry.slice(key.length + 1);
	try {
		return decodeURIComponent(value);
	} catch {
		return value;
	}
}

export function setManualLocaleCookie(locale: string): void {
	if (typeof document === 'undefined') return;
	const encoded = encodeURIComponent(locale);
	document.cookie = `${MANUAL_LOCALE_COOKIE_NAME}=${encoded}; path=/; max-age=31536000; SameSite=Lax`;
}

export function clearManualLocaleCookie(): void {
	if (typeof document === 'undefined') return;
	document.cookie = `${MANUAL_LOCALE_COOKIE_NAME}=; path=/; max-age=0; SameSite=Lax`;
}

export function getManualLocaleFromDocument(): string | undefined {
	if (typeof document === 'undefined') return undefined;
	return getCookieValue(document.cookie, MANUAL_LOCALE_COOKIE_NAME);
}
