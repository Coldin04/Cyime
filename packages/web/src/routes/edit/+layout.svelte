<script lang="ts">
	import RouteAuthGuard from '$lib/components/auth/RouteAuthGuard.svelte';

	let { children } = $props();
</script>

<RouteAuthGuard mode="required">
	{@render children()}
</RouteAuthGuard>
