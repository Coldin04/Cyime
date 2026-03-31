<script lang="ts">
	import RouteAuthGuard from '$lib/components/auth/RouteAuthGuard.svelte';

	let { children } = $props();
</script>

<RouteAuthGuard mode="optional">
	{@render children()}
</RouteAuthGuard>
