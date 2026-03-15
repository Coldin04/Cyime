import type { Handle } from '@sveltejs/kit';
import { paraglideMiddleware } from '$paraglide/server';
import { defineCustomServerStrategy, isLocale } from '$paraglide/runtime';
import { getCookieValue, MANUAL_LOCALE_COOKIE_NAME } from '$lib/paraglide/manual-locale-cookie';

defineCustomServerStrategy('custom-manual-cookie', {
	getLocale: (request) => {
		const manualLocale = getCookieValue(request?.headers.get('cookie'), MANUAL_LOCALE_COOKIE_NAME);
		if (manualLocale && isLocale(manualLocale)) return manualLocale;
		return undefined;
	}
});

// creating a handle to use the paraglide middleware
const paraglideHandle: Handle = ({ event, resolve }) => {
	// API 路由不参与页面语言中间件，避免给接口请求附加无意义的 i18n 处理。
	if (event.url.pathname.startsWith('/api/')) {
		return resolve(event);
	}

	return paraglideMiddleware(event.request, ({ request: localizedRequest, locale }) => {
		event.request = localizedRequest;
		return resolve(event, {
			transformPageChunk: ({ html }) => {
				return html.replace('%lang%', locale);
			}
		});
	});
};

export const handle: Handle = paraglideHandle;
