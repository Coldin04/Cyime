<script lang="ts">
	import { browser } from '$app/environment';
	import MagnifyingGlass from '~icons/ph/magnifying-glass';
	import Logo from '$lib/components/common/Logo.svelte';
	import UserMenuDropdown from '$lib/components/common/UserMenuDropdown.svelte';
	import GlobalSearchDialog from '$lib/components/common/GlobalSearchDialog.svelte';
	import * as m from '$paraglide/messages';

	let isSearchOpen = $state(false);
	const searchShortcutLabel = $derived(
		browser && /Mac|iPhone|iPad/i.test(window.navigator.platform) ? '⌘K' : 'Ctrl K'
	);

	function openSearch() {
		isSearchOpen = true;
	}

	function closeSearch() {
		isSearchOpen = false;
	}

	function handleGlobalKeydown(event: KeyboardEvent) {
		if (event.isComposing) {
			return;
		}
		if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
			event.preventDefault();
			openSearch();
		}
	}
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<nav
	class="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-black/10 bg-white/80 px-4 backdrop-blur-md dark:border-white/10 dark:bg-zinc-900/80"
>
	<div class="flex items-center gap-2">
		<Logo href="/workspace" labelClass="text-lg font-bold" />
	</div>

	<div class="flex items-center gap-4">
		<button
			class="relative grid h-8 w-8 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
			type="button"
			title={`${m.common_search_placeholder()} (${searchShortcutLabel})`}
			aria-label={`${m.common_search_placeholder()} (${searchShortcutLabel})`}
			onclick={openSearch}
		>
			<MagnifyingGlass class="h-5 w-5" />
			<span class="absolute -right-3 -top-1 hidden rounded border border-zinc-200 bg-white px-1 py-0 text-[9px] font-semibold leading-none text-zinc-500 shadow-sm sm:inline-flex dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400">
				{searchShortcutLabel}
			</span>
		</button>
		<UserMenuDropdown profileHref="/user" trashHref="/workspace/trash" showTrash={true} />
	</div>
</nav>

<GlobalSearchDialog open={isSearchOpen} onClose={closeSearch} />
