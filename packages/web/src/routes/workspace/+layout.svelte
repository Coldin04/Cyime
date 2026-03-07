<script lang="ts">
	import * as m from '$paraglide/messages';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import TopBar from '$lib/components/workspace/TopBar.svelte';
	import GreetingHeader from '$lib/components/workspace/GreetingHeader.svelte';
	import { workspaceContext } from '$lib/stores/workspace';

	// Route guard using $effect
	$effect(() => {
		// We only run this logic in the browser.
		if (browser) {
			// We wait for the auth store's loading process to complete.
			if (!$auth.loading && !$auth.token) {
				goto('/login', { replaceState: true });
			}
		}
	});
</script>

<!--
  This conditional rendering ensures a good user experience:
  1. If the auth state is loading, show a loading message.
  2. If loading is complete AND there is a token, show the actual page content.
  User info will be fetched on-demand by components that need it.
-->
{#if $auth.loading}
	<div class="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-gray-900">
		<p class="text-lg text-gray-600 dark:text-gray-300">{m.workspace_loading()}</p>
	</div>
{:else if $auth.token}
	<TopBar />
	<main class="max-w-5xl mx-auto px-4 sm:px-6 py-8">
		<GreetingHeader />
		<slot />
	</main>
{/if}
