import { auth } from '$lib/stores/auth';
import { resolveApiUrl } from '$lib/config/api';
import { get } from 'svelte/store';

// In-flight refresh promise. When N concurrent requests all see a 401, only
// the first triggers the refresh; the rest await the same promise and reuse
// the new token. The previous boolean-flag approach made the second-Nth request
// silently fall through and return the original 401, which surfaced as random
// UI errors during the refresh window.
let refreshPromise: Promise<string | null> | null = null;

function refreshTokenOnce(): Promise<string | null> {
	if (!refreshPromise) {
		refreshPromise = auth.refreshToken().finally(() => {
			refreshPromise = null;
		});
	}
	return refreshPromise;
}

/**
 * A custom fetch wrapper that automatically adds the Authorization header
 * and handles token refreshing and request retrying on 401 errors.
 * @param url The URL to fetch.
 * @param options The options for the fetch request.
 * @returns A Promise that resolves to the Response object.
 */
export async function apiFetch(url: string, options: RequestInit = {}): Promise<Response> {
	const resolvedUrl = resolveApiUrl(url);

	// Get the current token from the store.
	const token = get(auth).token;

	// Set up the headers.
	const headers = new Headers(options.headers);
	if (token) {
		headers.set('Authorization', `Bearer ${token}`);
	}
	options.headers = headers;
	if (options.credentials === undefined) {
		options.credentials = 'include';
	}

	// Make the initial request.
	let response = await fetch(resolvedUrl, options);

	// If the response is a 401 Unauthorized, share a single refresh attempt
	// across all concurrent callers and retry exactly once.
	if (response.status === 401) {
		try {
			const newAccessToken = await refreshTokenOnce();
			if (newAccessToken) {
				headers.set('Authorization', `Bearer ${newAccessToken}`);
				options.headers = headers;
				response = await fetch(resolvedUrl, options);
			}
		} catch (error) {
			// The refresh failed; auth store has already triggered logout. Return
			// the original 401 to the caller so the UI can react.
			console.error('Failed to retry request after token refresh.', error);
			return response;
		}
	}

	return response;
}
