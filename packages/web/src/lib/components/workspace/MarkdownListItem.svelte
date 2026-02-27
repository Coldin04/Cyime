<script lang="ts">
	import FileMd from '~icons/ph/file-md';
</script>

<a
	href="/workspace"
	class="group flex cursor-pointer items-center border-b border-zinc-200 px-4 py-3 transition-colors hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800/60"
>
	<!-- Name Column -->
	<div class="flex flex-1 items-center gap-3">
		<div class="relative h-6 w-6 flex-shrink-0">
			<!-- Checkbox -->
			<input
				type="checkbox"
				class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded border-zinc-400 opacity-0 transition-opacity group-hover:opacity-100"
			/>
			<!-- Icon -->
			<FileMd class="h-6 w-6 text-zinc-500 transition-opacity group-hover:opacity-0" />
		</div>
		<span class="font-medium text-zinc-800 dark:text-zinc-200"
			>My SvelteKit Notes.md</span
		>
	</div>

	<!-- Last Modified Column -->
	<div class="w-48 hidden text-sm text-zinc-600 dark:text-zinc-400 sm:block">
		1 hour ago
	</div>

	<!-- Owner Column -->
	<div class="w-32 hidden text-sm text-zinc-600 dark:text-zinc-400 md:block">
		You
	</div>
</a>
