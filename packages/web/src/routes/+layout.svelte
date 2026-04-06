<script lang="ts">
	import '$lib/paraglide-strategy.ts';
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { Toaster } from 'svelte-sonner';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { initThemeModeSync } from '$lib/theme/preference';
	import * as m from '$paraglide/messages';

	let { children } = $props();
	const canonicalUrl = $derived($page.url.href);
	const pageLanguage = $derived(
		typeof $page.data.languageTag === 'string'
			? $page.data.languageTag
			: $page.url.pathname.startsWith('/en')
				? 'en'
				: 'zh'
	);
	const ogLocale = $derived(pageLanguage.startsWith('en') ? 'en_US' : 'zh_CN');

	onMount(() => {
		return initThemeModeSync();
	});
</script>

<svelte:head>
	<link rel="icon" href="{favicon}" />
	<meta name="application-name" content={m.meta_site_name()} />
	<meta name="apple-mobile-web-app-title" content={m.meta_project_short_name()} />
	<meta name="author" content={m.meta_project_author()} />
	<meta name="keywords" content={m.meta_site_keywords()} />
	<meta name="description" content={m.meta_site_description()} />
	<meta name="theme-color" content="#14b8a6" />
	<meta property="og:site_name" content={m.meta_site_name()} />
	<meta property="og:locale" content={ogLocale} />
	<meta property="og:type" content="website" />
	<meta property="og:title" content={m.meta_site_title()} />
	<meta property="og:description" content={m.meta_site_description()} />
	<meta property="og:url" content={canonicalUrl} />
	<meta name="twitter:card" content="summary" />
	<meta name="twitter:title" content={m.meta_site_title()} />
	<meta name="twitter:description" content={m.meta_site_description()} />
</svelte:head>

<Toaster richColors closeButton />
{@render children()}
