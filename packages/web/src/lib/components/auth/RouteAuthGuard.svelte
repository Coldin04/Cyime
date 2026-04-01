<script lang="ts">
	import * as m from '$paraglide/messages';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';

	type Props = {
		mode?: 'required' | 'optional';
	};

	let { children, mode = 'required' }: Props & { children: import('svelte').Snippet } = $props();
	const requiresAuth = $derived(mode === 'required');

	$effect(() => {
		if (!browser) return;
		if (requiresAuth && !$auth.loading && !$auth.token) {
			goto('/login', { replaceState: true });
		}
	});
</script>

{#if $auth.loading}
	<div class="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-gray-900">
		<p class="text-lg text-gray-600 dark:text-gray-300">{m.workspace_loading()}</p>
	</div>
{:else if !requiresAuth || $auth.token}
	{@render children()}
{/if}
