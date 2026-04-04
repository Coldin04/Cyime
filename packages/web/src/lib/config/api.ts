import { env } from '$env/dynamic/public';

const rawApiBaseUrl = (env.PUBLIC_API_BASE_URL ?? '').trim();

if (!rawApiBaseUrl) {
	throw new Error('PUBLIC_API_BASE_URL is required');
}

function trimTrailingSlash(value: string): string {
	return value.endsWith('/') ? value.slice(0, -1) : value;
}

export const apiBaseUrl = trimTrailingSlash(rawApiBaseUrl);

export function resolveApiUrl(path: string): string {
	if (!path.startsWith('/')) {
		return path;
	}

	return `${apiBaseUrl}${path}`;
}
