<script lang="ts">
	import FileMd from '~icons/ph/file-md';
</script>

<a
	href="/workspace"
	class="group flex cursor-pointer items-center border-b border-zinc-200 px-4 py-3 transition-colors hover:bg-gradient-to-r hover:from-blue-50/50 hover:to-transparent dark:border-zinc-700 dark:hover:bg-zinc-800/60"
>
	<!-- Name Column -->
	<div class="flex flex-1 items-center gap-3">
		<!-- Checkbox -->
		<input
			type="checkbox"
			disabled
			class="h-4 w-4 rounded border-zinc-400 opacity-0 transition-opacity group-hover:opacity-100 group-hover:enabled:cursor-pointer"
		/>
		<!-- Icon -->
		<FileMd class="h-6 w-6 text-blue-500 dark:text-blue-400" />
		<span class="font-normal text-zinc-800 dark:text-zinc-200"
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
