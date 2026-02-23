<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';

	// This reactive block is the core of our route guard.
	// It automatically re-runs whenever the value of `$auth` changes.
	$: {
		// We only run this logic in the browser.
		if (browser) {
			// We wait for the auth store's loading process to complete.
			// This prevents a premature redirect before the app has had a chance
			// to establish a session (e.g., in future versions with localStorage).
			if (!$auth.loading && !$auth.user) {
				// If authentication is NOT loading, and we have NO user,
				// it means the user is definitively logged out. Redirect them.
				goto('/login', { replaceState: true });
			}
		}
	}
</script>

<!-- 
  This conditional rendering ensures a good user experience:
  1. If the auth state is loading, show a loading message.
  2. If loading is complete AND there is a user, show the actual page content.
  The case where loading is complete and there is no user is handled by the redirect above,
  so the user will never see a blank page.
-->
{#if $auth.loading}
	<div class="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-gray-900">
		<p class="text-lg text-gray-600 dark:text-gray-300">正在验证身份...</p>
	</div>
{:else if $auth.user}
	<slot />
{/if}
