import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

export type User = {
	id: string;
	displayName: string | null;
	email: string | null;
	avatarUrl: string | null;
};

type AuthState = {
	token: string | null;
	user: User | null;
	loading: boolean;
};

let refreshTimerId: NodeJS.Timeout | null = null;

function parseJwt(token: string): { exp?: number } {
	try {
		const base64Url = token.split('.')[1];
		const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
		const jsonPayload = JSON.parse(atob(base64));
		return { exp: jsonPayload.exp };
	} catch (e) {
		return {};
	}
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>({
		token: null,
		user: null,
		loading: true
	});

	async function refreshToken() {
		console.log('Attempting to refresh token...');
		try {
			const response = await fetch('/api/v1/auth/refresh', {
				method: 'POST'
			});
			if (!response.ok) throw new Error('Refresh failed');

			const { accessToken: newAccessToken } = await response.json();
			
			update(state => ({ ...state, token: newAccessToken }));
			scheduleRefresh(newAccessToken);
			console.log('Token refreshed successfully.');
			return newAccessToken; // Return the new token on success

		} catch (error) {
			console.error('Could not refresh token:', error);
			logout(); // If refresh fails, the session is over.
			throw error; // Propagate the error
		}
	}

	function scheduleRefresh(token: string) {
		if (refreshTimerId) {
			clearTimeout(refreshTimerId);
		}

		const { exp } = parseJwt(token);
		if (!exp) return;

		const expiresAt = exp * 1000;
		const now = Date.now();
		const expiresIn = expiresAt - now;

		// Schedule refresh for 85% of the token's remaining lifetime.
		const timeout = expiresIn * 0.85;

		if (timeout > 0) {
			refreshTimerId = setTimeout(refreshToken, timeout);
		}
	}

	async function init() {
		if (!browser) {
			update((state) => ({ ...state, loading: false }));
			return;
		}
		// In a future step with persistence, we would check localStorage for a token here.
		update((state) => ({ ...state, loading: false }));
	}

	async function loginAndFetchUser(token: string) {
		const { exp } = parseJwt(token);
		if (!exp || exp * 1000 < Date.now()) {
			logout();
			return;
		}

		try {
			const response = await fetch('/api/v1/user/me', {
				headers: { Authorization: `Bearer ${token}` }
			});
			if (!response.ok) throw new Error('Failed to fetch user profile');
			
			const user: User = await response.json();
			set({ token, user, loading: false });
			scheduleRefresh(token); // Schedule the first refresh on successful login.
		
		} catch (error) {
			console.error('Failed to log in:', error);
			logout();
		}
	}

	async function logout() {
		if (refreshTimerId) {
			clearTimeout(refreshTimerId);
			refreshTimerId = null;
		}

		try {
			// Inform the backend to revoke the refresh token.
			const response = await fetch('/api/v1/auth/logout', {
				method: 'POST'
			});
			if (!response.ok) {
				// We can log this error, but we still want to clear the local state.
				console.error('Backend logout failed:', response.statusText);
			}
		} catch (error) {
			// Also log network errors etc.
			console.error('Error during logout API call:', error);
		} finally {
			// Always clear the local state to log the user out on the frontend.
			set({ token: null, user: null, loading: false });
		}
	}

	return {
		subscribe,
		init,
		loginAndFetchUser,
		logout,
		refreshToken // Expose the refresh function
	};
}

export const auth = createAuthStore();

auth.init();
