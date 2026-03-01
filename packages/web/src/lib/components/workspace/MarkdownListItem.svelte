<script lang="ts">
	import FileMd from '~icons/ph/file-md';
	import type { FileItem } from '$lib/api/workspace';

	let {
		item,
		selectedItems
	}: {
		item: FileItem;
		selectedItems: Set<string>;
	} = $props();

	function formatRelativeTime(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

		if (diffInSeconds < 60) {
			return '刚刚';
		} else if (diffInSeconds < 3600) {
			const minutes = Math.floor(diffInSeconds / 60);
			return `${minutes} 分钟前`;
		} else if (diffInSeconds < 86400) {
			const hours = Math.floor(diffInSeconds / 3600);
			return `${hours} 小时前`;
		} else if (diffInSeconds < 604800) {
			const days = Math.floor(diffInSeconds / 86400);
			return `${days} 天前`;
		} else {
			return date.toLocaleDateString('zh-CN', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		}
	}

	function toggleSelection() {
		if (selectedItems.has(item.id)) {
			selectedItems.delete(item.id);
		} else {
			selectedItems.add(item.id);
		}
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === ' ' || event.key === 'Enter') {
			event.preventDefault();
			toggleSelection();
		}
	}
</script>

<div
	role="button"
	tabindex="0"
	class="group flex cursor-pointer items-center border-b border-zinc-200 px-4 py-3 transition-colors hover:bg-gradient-to-r hover:from-blue-50/50 hover:to-transparent dark:border-zinc-700 dark:hover:bg-zinc-800/60 {selectedItems.has(
		item.id
	)
		? 'bg-blue-50 dark:bg-blue-900/30'
		: ''}"
	onclick={toggleSelection}
	onkeydown={handleKeyDown}
>
	<!-- Name Column -->
	<div class="flex flex-1 items-center gap-3">
		<!-- Checkbox -->
		<input
			type="checkbox"
			class="h-4 w-4 rounded border-zinc-400 opacity-0 transition-opacity group-hover:opacity-100"
			checked={selectedItems.has(item.id)}
			onclick={(e) => e.stopPropagation()}
			onchange={toggleSelection}
		/>
		<FileMd class="h-5 w-5 text-blue-500 dark:text-blue-400" />
		<a
			href="/edit/md/{item.id}"
			class="font-normal text-zinc-800 dark:text-zinc-200"
			onclick={(e) => e.stopPropagation()}
		>
			{item.title}
		</a>
	</div>

	<!-- Last Modified Column -->
	<div class="w-48 hidden text-sm text-zinc-600 dark:text-zinc-400 sm:block">
		{formatRelativeTime(item.updatedAt)}
	</div>

	<!-- Owner Column -->
	<div class="w-32 hidden text-sm text-zinc-600 dark:text-zinc-400 md:block">
		{item.creator.displayName || 'You'}
	</div>
</div>
