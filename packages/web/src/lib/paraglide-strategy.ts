import { defineCustomClientStrategy } from '../paraglide/runtime.js';
import { browser } from '$app/environment';

if (browser) {


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
			const match = document.cookie.match(/CYIMEWRITE_LOCALE=([^;]+)/);
			return match ? match[1] : undefined;
		},

		/**
		 * Sets the 'PARAGLIDE_LOCALE' cookie, but skips the very first call.
		 * @param {string} locale - The locale to set.
		 */
		setLocale: (locale) => {
			// The locale is now managed manually by the application.
			// This function will not automatically write the cookie.
			// Developers must manually set `document.cookie = `CYIMEWRITE_LOCALE=${locale};path=/;max-age=31536000;SameSite=Lax`
			// in their application code when a persistent language change is desired.
		}
	});
}
