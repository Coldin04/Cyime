<script lang="ts">
	import type { BreadcrumbItem } from '$lib/stores/workspace';
	import Home from '~icons/ph/house';
	import CaretRight from '~icons/ph/caret-right';
	import * as m from '$paraglide/messages';

	import { browser } from '$app/environment';

	const {
		items = [],
		onNavigate
	}: {
		items: BreadcrumbItem[];
		onNavigate?: (id: string | null) => void;
	} = $props();

	const MAX_BREADCRUMBS = 4; // Max items to show before truncating with '...' on large screens
	let isSmallScreen = $state(false);

	// This effect runs only on the client to detect screen size
	if (browser) {
		$effect(() => {
			const media = window.matchMedia('(max-width: 640px)'); // Tailwind's `sm` breakpoint

			function update(e: MediaQueryListEvent | MediaQueryList) {
				isSmallScreen = e.matches;
			}

			media.addEventListener('change', update);
			update(media); // Initial check

			return () => {
				media.removeEventListener('change', update);
			};
		});
	}

	let visibleCrumbs: BreadcrumbItem[] = $derived.by(() => {
		// Special, more aggressive truncation for small screens
		if (isSmallScreen && items.length > 2) {
			return [{ id: '...', name: '...' }, items[items.length - 1]];
		}

		// Default truncation for larger screens
		if (items.length <= MAX_BREADCRUMBS) {
			return items;
		}
		// Show first, ..., last two items for paths longer than 4
		return [items[0], { id: '...', name: '...' }, ...items.slice(-2)];
	});
</script>

<nav class="flex items-center text-sm font-medium text-zinc-600 dark:text-zinc-400">
	<button
		onclick={() => onNavigate?.(null)}
		class="flex items-center gap-1 rounded-md px-2 py-1 transition-colors hover:bg-zinc-200 hover:text-zinc-800 dark:hover:bg-zinc-700 dark:hover:text-zinc-200"
	>
		<Home class="h-4 w-4" />
		<span>{m.breadcrumb_all_files()}</span>
	</button>

	{#if items.length > 0}
		<CaretRight class="h-4 w-4 text-zinc-400" />
	{/if}

	{#each visibleCrumbs as crumb, i (crumb.id)}
		<div class="flex items-center">
			{#if crumb.name === '...'}
				<span class="px-2">...</span>
			{:else}
				<button
					onclick={() => onNavigate?.(crumb.id)}
					class="block max-w-[120px] truncate rounded-md px-2 py-1 transition-colors hover:bg-zinc-200 hover:text-zinc-800 dark:hover:bg-zinc-700 dark:hover:text-zinc-200"
				>
					{crumb.name}
				</button>
			{/if}

			{#if i < visibleCrumbs.length - 1}
				<CaretRight class="h-4 w-4 text-zinc-400" />
			{/if}
		</div>
	{/each}
</nav>

