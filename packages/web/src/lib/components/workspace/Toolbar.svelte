<script lang="ts">
	import CaretRight from '~icons/ph/caret-right';
	import Plus from '~icons/ph/plus';
</script>

<div class="flex items-center justify-between pt-4 pb-8">
	<!-- Breadcrumbs -->
	<nav aria-label="Breadcrumb" class="flex items-center text-zinc-500 dark:text-zinc-400">
		<a href="/workspace" class="hover:underline">Workspace</a>
		<CaretRight class="mx-1 h-4 w-4" />
		<a class="hover:underline">Learning</a>
		<CaretRight class="mx-1 h-4 w-4" />
		<span class="font-medium text-zinc-800 dark:text-zinc-200">Svelte</span>
	</nav>

	<!-- Action Button -->
	<button
		class="inline-flex items-center justify-center gap-2 rounded-lg bg-riptide-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-riptide-700 active:bg-riptide-800 disabled:opacity-50"
	>
		<Plus class="h-4 w-4" />
		<span>新建文档</span>
	</button>
</div>
