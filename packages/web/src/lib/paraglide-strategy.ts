import { defineCustomClientStrategy } from '../paraglide/runtime.js';
import { browser } from '$app/environment';

if (browser) {
	let isInitialSet = true;

	/**
	 * A custom cookie strategy that only writes the cookie after the initial, automatic
	 * `setLocale` call. This prevents the browser's preferred language from being
	 * immediately persisted on a user's first visit, allowing for proper language
	 * detection on subsequent visits until a language is explicitly chosen.
	 */
	defineCustomClientStrategy('custom-manual-cookie', {
		/**
		 * Reads the locale from the 'PARAGLIDE_LOCALE' cookie.
		 * @returns The locale string or `undefined` if the cookie is not set.
		 */
		getLocale: () => {
			const match = document.cookie.match(/PARAGLIDE_LOCALE=([^;]+)/);
			return match ? match[1] : undefined;
		},

		/**
		 * Sets the 'PARAGLIDE_LOCALE' cookie, but skips the very first call.
		 * @param {string} locale - The locale to set.
		 */
		setLocale: (locale) => {
			if (isInitialSet) {
				isInitialSet = false;
				// On the first, automatic `setLocale` call that happens on page load,
				// we do nothing. This allows the `preferredLanguage` strategy to work
				// without immediately persisting the detected locale.
				return;
			}
			// For all subsequent, user-initiated calls, we write the cookie.
			// This persists the user's explicit language choice.
			document.cookie = `PARAGLIDE_LOCALE=${locale};path=/;max-age=31536000;SameSite=Lax`;
		}
	});
}
