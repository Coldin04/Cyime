<script lang="ts">
	import * as m from '$paraglide/messages';
	import { auth } from '$lib/stores/auth';

	function handleLogout() {
		auth.logout();
		// No need to call goto('/login') here.
		// The +layout.svelte guard will react to the store change and handle the redirect.
	}
</script>

<div class="p-8 text-gray-800 dark:text-gray-50">
	<div class="flex items-center justify-between">
		<h1 class="text-4xl font-bold">{m.workspace_title()}</h1>
		<button
			on:click={handleLogout}
			class="rounded-md bg-gray-200 px-4 py-2 text-sm font-medium text-gray-800 hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
		>
			Logout
		</button>
	</div>
	<p class="mt-4">{m.workspace_document_list_under_construction()}</p>
</div>
