<script lang="ts">
	import FileMd from '~icons/ph/file-md';
	import type { FileItem } from '$lib/api/workspace';
	import DotsThreeVertical from '~icons/ph/dots-three-vertical';

	let {
		item,
		selectedItems,
		bulkMode = false,
		onToggle
	}: {
		item: FileItem;
		selectedItems: { [key:string]: boolean };
		bulkMode?: boolean;
		onToggle: (id: string) => void;
	} = $props();

	const isSelected = $derived(!!selectedItems[item.id]);
	const checkboxClasses = $derived(
		`h-4 w-4 rounded border-zinc-400 transition-opacity dark:border-zinc-600 ${
			bulkMode || isSelected ? 'opacity-100' : 'opacity-0'
		}`
	);

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

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === ' ' || event.key === 'Enter') {
			event.preventDefault();
			onToggle(item.id);
		}
	}
</script>

<div
	role="button"
	tabindex="0"
	class="group flex cursor-pointer items-center justify-between border-b border-zinc-200 px-4 py-3 transition-colors hover:bg-gradient-to-r hover:from-blue-50/50 hover:to-transparent dark:border-zinc-700 dark:hover:bg-zinc-800/60 {isSelected
		? 'bg-blue-50 dark:bg-blue-900/30'
		: ''}"
	onclick={() => onToggle(item.id)}
	onkeydown={handleKeyDown}
>
	<!-- Left Side: Name -->
	<div class="flex min-w-0 items-center gap-3 pr-4">
		<input
			type="checkbox"
			class={checkboxClasses}
			checked={isSelected}
			onclick={(e) => e.stopPropagation()}
			onchange={() => onToggle(item.id)}
		/>
		<FileMd class="h-5 w-5 flex-shrink-0 text-blue-500 dark:text-blue-400" />
		<a
			href="/edit/md/{item.id}"
			class="truncate font-normal text-zinc-800 dark:text-zinc-200"
			onclick={(e) => e.stopPropagation()}
		>
			{item.title}
		</a>
	</div>

	<!-- Right Side: Metadata -->
	<div class="flex flex-shrink-0 items-center justify-end gap-x-4 sm:gap-x-6">
		<div class="hidden w-28 text-right text-sm text-zinc-600 dark:text-zinc-400 sm:block">
			{formatRelativeTime(item.updatedAt)}
		</div>
		<div class="hidden w-24 text-right text-sm text-zinc-600 dark:text-zinc-400 md:block pr-0.5">
			{item.creator.displayName || 'You'}
		</div>
		<div class="w-10 flex justify-center">
			<button
				class="rounded-full p-2 text-zinc-500 transition-colors hover:bg-zinc-200 dark:text-zinc-400 dark:hover:bg-zinc-700"
				onclick={(e) => {
					e.stopPropagation();
					// eslint-disable-next-line no-console
					console.log('More options for', item.id);
				}}
			>
				<DotsThreeVertical class="h-5 w-5" />
			</button>
		</div>
	</div>
</div>
