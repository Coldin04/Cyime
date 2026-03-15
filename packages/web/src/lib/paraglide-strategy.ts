import { defineCustomClientStrategy, isLocale } from '../paraglide/runtime.js';
import { browser } from '$app/environment';
import { getCookieValue, MANUAL_LOCALE_COOKIE_NAME } from '$lib/paraglide/manual-locale-cookie';

if (browser) {

	/**
	 * A custom cookie strategy that only reads a manually-managed cookie.
	 * It intentionally does not persist cookie on `setLocale`.
	 */
	defineCustomClientStrategy('custom-manual-cookie', {
		/**
		 * Reads locale from manually managed cookie `cyime-locale`.
		 * @returns The locale string or `undefined` if the cookie is not set.
		 */
		getLocale: () => {
			const manualLocale = getCookieValue(document.cookie, MANUAL_LOCALE_COOKIE_NAME);
			return manualLocale && isLocale(manualLocale) ? manualLocale : undefined;
		},

		/**
		 * Intentionally no-op:
		 * locale cookie is written by manual app logic only.
		 * @param {string} locale - The locale to set.
		 */
		setLocale: (_locale) => {
			// no-op by design
		}
	});
}
