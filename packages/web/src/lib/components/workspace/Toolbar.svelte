<script lang="ts">
	import Trash from '~icons/ph/trash';
	import X from '~icons/ph/x';
	import Breadcrumb from './Breadcrumb.svelte';
	import { breadcrumbItems } from '$lib/stores/workspace';
	import * as m from '$paraglide/messages';
	import ArrowRight from '~icons/ph/arrow-right';

	const {
		bulkMode = false,
		selectedItemsCount = 0,
		onToggleBulk,
		onBulkDelete,
		onBulkMove,
		onNavigate
	}: {
		bulkMode?: boolean;
		selectedItemsCount?: number;
		onToggleBulk: () => void;
		onBulkDelete: () => void;
		onBulkMove: () => void;
		onNavigate?: (id: string | null) => void;
	} = $props();
</script>

<div class="flex items-center justify-between">
	<div class="min-w-0 flex-1">
		<Breadcrumb onNavigate={onNavigate} items={$breadcrumbItems} />
	</div>

	{#if bulkMode}
		<!-- Bulk Mode Actions -->
		<div class="flex items-center gap-2">
			<span class="text-sm text-zinc-600 dark:text-zinc-400">
				{m.toolbar_selected_count({ count: selectedItemsCount })}
			</span>
			<button
				onclick={onBulkMove}
				class="inline-flex h-10 items-center gap-2 rounded-lg bg-blue-600 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-blue-700"
			>
				<ArrowRight class="h-4 w-4" />
				<span class="hidden sm:inline">{m.common_move()}</span>
			</button>
			<button
				onclick={onBulkDelete}
				class="inline-flex h-10 items-center gap-2 rounded-lg bg-red-500 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-red-600"
			>
				<Trash class="h-4 w-4" />
				<span class="hidden sm:inline">{m.common_delete()}</span>
			</button>
			<button
				onclick={onToggleBulk}
				class="inline-flex h-10 items-center justify-center rounded-lg border border-zinc-300 bg-white px-3 text-sm font-semibold text-zinc-700 shadow-sm transition-colors hover:bg-zinc-50 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
			>
				<X class="h-4 w-4" />
			</button>
		</div>
	{/if}
</div>

