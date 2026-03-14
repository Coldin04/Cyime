import type { Handle } from '@sveltejs/kit';
import { paraglideMiddleware } from '$paraglide/server';
import { defineCustomServerStrategy } from '$paraglide/runtime';
import { getCookieValue, MANUAL_LOCALE_COOKIE_NAME } from '$lib/i18n/manual-locale-cookie';

defineCustomServerStrategy('custom-manual-cookie', {
	getLocale: (request) => getCookieValue(request?.headers.get('cookie'), MANUAL_LOCALE_COOKIE_NAME)
});

// creating a handle to use the paraglide middleware
const paraglideHandle: Handle = ({ event, resolve }) =>
	paraglideMiddleware(event.request, ({ request: localizedRequest, locale }) => {
		event.request = localizedRequest;
		return resolve(event, {
			transformPageChunk: ({ html }) => {
				return html.replace('%lang%', locale);
			}
		});
	});

export const handle: Handle = paraglideHandle;
