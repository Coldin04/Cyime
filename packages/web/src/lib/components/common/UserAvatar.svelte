<script lang="ts">
	import * as m from '$paraglide/messages';

	interface Props {
		name?: string | null;
		avatarUrl?: string | null;
		size?: number;
		className?: string;
	}

	let { name = null, avatarUrl = null, size = 64, className = '' }: Props = $props();

	let loadFailed = $state(false);
	let loaded = $state(false);
	let imgEl = $state<HTMLImageElement | null>(null);
	const normalizedUrl = $derived((avatarUrl || '').trim());
	const displayName = $derived((name || '').trim());
	const fallbackInitial = $derived((displayName || m.common_user()).charAt(0).toUpperCase());

	$effect(() => {
		const _url = normalizedUrl;
		loadFailed = false;
		loaded = false;
	});

	$effect(() => {
		if (imgEl && imgEl.complete && imgEl.naturalWidth > 0) {
			loaded = true;
		}
	});
</script>

<div
	class={`relative grid place-content-center overflow-hidden rounded-full bg-riptide-100 dark:bg-riptide-900 ${className}`}
	style={`width:${size}px;height:${size}px;`}
>
	{#if normalizedUrl && !loadFailed}
		{#if !loaded}
			<div
				class="absolute inset-0 animate-pulse bg-riptide-200/80 dark:bg-riptide-800/70"
				aria-hidden="true"
			></div>
		{/if}
		<img
			bind:this={imgEl}
			src={normalizedUrl}
			alt={m.greeting_avatar_alt({ name: displayName || m.common_user() })}
			class="h-full w-full rounded-full object-cover transition-opacity duration-200"
			class:opacity-0={!loaded}
			class:opacity-100={loaded}
			decoding="async"
			fetchpriority="low"
			referrerpolicy="no-referrer"
			onload={() => {
				loaded = true;
			}}
			onerror={() => {
				loadFailed = true;
			}}
		/>
	{:else}
		<span class="text-xl font-bold text-riptide-600 dark:text-riptide-300">{fallbackInitial}</span>
	{/if}
</div>
