<script lang="ts">
	import * as m from '$paraglide/messages';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	
	// This reactive block is the core of our route guard.
	// It automatically re-runs whenever the value of `$auth` changes.
	$: {
		// We only run this logic in the browser.
		if (browser) {
			// We wait for the auth store's loading process to complete.
			if (!$auth.loading && !$auth.token) {
				// If loading is complete and there's no token, redirect to login.
				goto('/login', { replaceState: true });
			}
		}
	}
</script>

<!--
  This conditional rendering ensures a good user experience:
  1. If the auth state is loading, show a loading message.
  2. If loading is complete AND there is a token, show the actual page content.
-->
{#if $auth.loading}
	<div class="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-gray-900">
		<p class="text-lg text-gray-600 dark:text-gray-300">{m.workspace_loading()}</p>
	</div>
{:else if $auth.token}
	<slot />
{/if}
