<script lang="ts">
	import * as m from '$paraglide/messages';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';

	let { children } = $props();

	$effect(() => {
		if (!browser) {
			return;
		}
		if (!$auth.loading && !$auth.token) {
			goto('/login', { replaceState: true });
		}
	});
</script>

{#if $auth.loading}
	<div class="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-gray-900">
		<p class="text-lg text-gray-600 dark:text-gray-300">{m.workspace_loading()}</p>
	</div>
{:else if $auth.token}
	{@render children()}
{/if}
