<script lang="ts">
	import { onMount } from 'svelte';
	import * as m from '$paraglide/messages';
	import { getUserOverview, type UserOverview } from '$lib/api/user';

	let overview = $state<UserOverview | null>(null);
	let loading = $state(true);
	let errorMessage = $state('');

	onMount(() => {
		void loadOverview();
	});

	async function loadOverview() {
		loading = true;
		errorMessage = '';
		try {
			overview = await getUserOverview();
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : m.user_overview_load_failed();
		} finally {
			loading = false;
		}
	}

	function formatLimit(limit: number | null, unlimited: boolean): string {
		if (unlimited || limit === null) return m.user_overview_unlimited();
		return String(limit);
	}
</script>

<div class="space-y-4">
	{#if loading}
		<div class="rounded-xl border border-dashed border-zinc-200 px-4 py-6 text-sm text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
			{m.common_loading()}
		</div>
	{:else if errorMessage}
		<div class="rounded-xl border border-rose-200 bg-rose-50 px-4 py-6 text-sm text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/20 dark:text-rose-300">
			{errorMessage}
		</div>
	{:else if overview}
		<div class="grid gap-4 md:grid-cols-3">
			<div class="rounded-xl border border-zinc-200 p-5 dark:border-zinc-800">
				<p class="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">{m.user_overview_active_documents()}</p>
				<p class="mt-2 text-3xl font-semibold text-zinc-900 dark:text-zinc-100">{overview.activeDocumentCount}</p>
				<p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">{m.user_overview_active_documents_hint()}</p>
			</div>

			<div class="rounded-xl border border-zinc-200 p-5 dark:border-zinc-800">
				<p class="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">{m.user_overview_trashed_documents()}</p>
				<p class="mt-2 text-3xl font-semibold text-zinc-900 dark:text-zinc-100">{overview.trashedDocumentCount}</p>
				<p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">{m.user_overview_trashed_documents_hint()}</p>
			</div>

			<div class="rounded-xl border border-zinc-200 p-5 dark:border-zinc-800">
				<p class="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">{m.user_overview_document_limit()}</p>
				<p class="mt-2 text-3xl font-semibold text-zinc-900 dark:text-zinc-100">{formatLimit(overview.documentLimit, overview.unlimited)}</p>
				<p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
					{#if overview.unlimited}
						{m.user_overview_limit_unlimited_hint()}
					{:else}
						{m.user_overview_limit_usage({
							count: String(overview.activeDocumentCount),
							limit: String(overview.documentLimit ?? 0)
						})}
					{/if}
				</p>
			</div>
		</div>

	{/if}
</div>
