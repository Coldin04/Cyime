<script lang="ts">
	import RouteAuthGuard from '$lib/components/auth/RouteAuthGuard.svelte';
	import TopBar from '$lib/components/workspace/TopBar.svelte';

	let { children } = $props();
</script>

<RouteAuthGuard mode="required">
	<TopBar />
	<main class="max-w-5xl mx-auto px-4 sm:px-6 py-8">
		{@render children()}
	</main>
</RouteAuthGuard>
