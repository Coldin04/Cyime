import { auth } from '$lib/stores/auth';
import { resolveApiUrl } from '$lib/config/api';
import { get } from 'svelte/store';

// A simple flag to prevent multiple concurrent refresh attempts.
// A more robust solution would use a promise-based lock.
let isRefreshing = false;

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

	// If the response is a 401 Unauthorized, and we haven't already started a refresh,
	// try to refresh the token and retry the request.
	if (response.status === 401 && !isRefreshing) {
		isRefreshing = true;
		try {
			// Attempt to refresh the token. The auth store handles the actual API call.
			// If this fails, it will throw an error and the user will be logged out by the store.
			const newAccessToken = await auth.refreshToken();

			// If refresh was successful, update the header with the new token...
			if (newAccessToken) {
				headers.set('Authorization', `Bearer ${newAccessToken}`);
				options.headers = headers;

				// ...and retry the original request.
				console.log('Retrying original request with new token.');
				response = await fetch(resolvedUrl, options);
			}
		} catch (error) {
			// The refresh failed, the auth store will handle logout.
			// We just return the original 401 response.
			console.error('Failed to retry request after token refresh.', error);
			return response;
		} finally {
			// Reset the flag regardless of outcome.
			isRefreshing = false;
		}
	}

	return response;
}
