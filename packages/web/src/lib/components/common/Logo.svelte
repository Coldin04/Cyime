<script lang="ts">
	let {
		href = '/',
		labelClass = 'text-lg font-bold',
		class: className = ''
	}: {
		href?: string;
		labelClass?: string;
		class?: string;
	} = $props();
</script>

<a
	href={href}
	class={`inline-flex items-center gap-2 transition-opacity hover:opacity-90 ${className}`}
	aria-label="Cyime"
>
	<span class={`bg-gradient-to-r from-cyan-300 to-sky-400 bg-clip-text text-transparent ${labelClass}`}>
		Cyime
	</span>
</a>
