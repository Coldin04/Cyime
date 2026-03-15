<script lang="ts">
	import '$lib/paraglide-strategy.ts';
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { Toaster } from 'svelte-sonner';
	import { onMount } from 'svelte';
	import { initThemeModeSync } from '$lib/theme/preference';

	let { children } = $props();

	onMount(() => {
		return initThemeModeSync();
	});
</script>

<svelte:head>
	<link rel="icon" href="{favicon}" />
</svelte:head>

<Toaster richColors closeButton />
{@render children()}
