<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import { page } from '$app/stores';

	onMount(async () => {
		// This component's only job is to capture the token from the URL,
		// hand it off to the auth store, and then redirect.
		const token = new URLSearchParams($page.url.hash.substring(1)).get('token');

		if (token) {
			// The store now handles the logic of fetching the user and setting the state.
			await auth.loginAndFetchUser(token);
			// After the store is updated, redirect to the main workspace.
			goto('/workspace', { replaceState: true });
		} else {
			// If no token is found, it's an invalid callback. Go back to login.
			goto('/login', { replaceState: true });
		}
	});
</script>

<div class="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-gray-900">
	<div class="text-center">
		<h1 class="text-xl font-semibold text-gray-700 dark:text-gray-200">正在完成登录...</h1>
		<p class="mt-2 text-sm text-gray-500">请稍候，我们将带您前往工作区。</p>
	</div>
</div>
