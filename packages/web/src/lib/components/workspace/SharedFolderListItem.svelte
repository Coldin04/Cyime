<script lang="ts">
	import UsersThree from '~icons/ph/users-three';
	import * as m from '$paraglide/messages';
	import { goto } from '$app/navigation';

	const {
		onNavigate
	}: {
		onNavigate?: () => void;
	} = $props();

	function handleClick() {
		onNavigate?.();
		goto('/workspace/shared');
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === ' ' || event.key === 'Enter') {
			event.preventDefault();
			handleClick();
		}
	}
</script>

<div
	role="button"
	tabindex="0"
	class="group flex cursor-pointer items-center justify-between border-b border-zinc-200 px-4 py-3 transition-colors hover:bg-gradient-to-r hover:from-sky-50/50 hover:to-transparent dark:border-zinc-700 dark:hover:bg-none dark:hover:bg-zinc-800/60"
	onclick={handleClick}
	onkeydown={handleKeyDown}
>
	<div class="flex min-w-0 items-center gap-3 pr-4">
		<div class="h-4 w-4 flex-shrink-0" aria-hidden="true"></div>
		<UsersThree class="h-5 w-5 flex-shrink-0 text-sky-500 dark:text-sky-400" />
		<span class="truncate font-normal text-zinc-800 dark:text-zinc-200">{m.workspace_shared_with_me_folder()}</span>
	</div>
	<div class="flex flex-shrink-0 items-center justify-end gap-x-4 sm:gap-x-6" aria-hidden="true">
		<div class="hidden w-28 sm:block"></div>
		<div class="hidden w-24 md:block pr-0.5"></div>
		<div class="w-10"></div>
	</div>
</div>
