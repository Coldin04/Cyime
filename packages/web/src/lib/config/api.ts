import { PUBLIC_API_BASE_URL } from '$env/static/public';

const rawApiBaseUrl = (PUBLIC_API_BASE_URL ?? '').trim();

function trimTrailingSlash(value: string): string {
	return value.endsWith('/') ? value.slice(0, -1) : value;
}

export const apiBaseUrl = rawApiBaseUrl ? trimTrailingSlash(rawApiBaseUrl) : '';

export function resolveApiUrl(path: string): string {
	if (!path.startsWith('/')) {
		return path;
	}

	return apiBaseUrl ? `${apiBaseUrl}${path}` : path;
}
