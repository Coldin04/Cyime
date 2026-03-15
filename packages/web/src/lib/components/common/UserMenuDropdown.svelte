<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import { clickOutside } from '$lib/actions/clickOutside';
	import * as m from '$paraglide/messages';
	import SignOut from '~icons/ph/sign-out';
	import Trash from '~icons/ph/trash';
	import UserAvatar from '$lib/components/common/UserAvatar.svelte';

	interface Props {
		profileHref?: string;
		trashHref?: string;
		showTrash?: boolean;
	}

	let { profileHref = '/user', trashHref = '/workspace/trash', showTrash = true }: Props = $props();

	let showMenu = $state(false);

	function toggleUserMenu() {
		showMenu = !showMenu;
	}

	function closeUserMenu() {
		showMenu = false;
	}

	function handleLogout() {
		auth.logout();
		showMenu = false;
	}

	afterNavigate(() => {
		closeUserMenu();
	});
</script>

<div
	class="relative"
	use:clickOutside={{
		enabled: showMenu,
		handler: closeUserMenu
	}}
>
	<button
		type="button"
		onclick={toggleUserMenu}
		class="flex h-9 w-9 items-center justify-center rounded-full border border-zinc-200 p-0 text-left transition-colors hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800 sm:h-auto sm:w-auto sm:justify-start sm:gap-2 sm:px-2 sm:py-1"
	>
		<UserAvatar size={28} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
		<div class="hidden min-w-0 sm:block">
			<p class="truncate text-xs font-semibold leading-tight text-zinc-800 dark:text-zinc-100">
				{$auth.user?.displayName || m.common_user()}
			</p>
			<p class="truncate text-[11px] leading-tight text-zinc-500 dark:text-zinc-400">
				{$auth.user?.email || 'No email'}
			</p>
		</div>
	</button>

	{#if showMenu}
		<div
			class="absolute top-full right-0 z-10 mt-2 w-56 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-zinc-800 dark:ring-zinc-700"
		>
			<a
				href={profileHref}
				onclick={closeUserMenu}
				class="flex items-center gap-3 px-4 py-3 text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
			>
				<UserAvatar size={36} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
				<div class="min-w-0">
					<p class="truncate text-sm font-semibold leading-tight">
						{$auth.user?.displayName || m.common_user()}
					</p>
					<p class="truncate text-xs leading-tight text-zinc-500 dark:text-zinc-400">
						{$auth.user?.email || 'No email'}
					</p>
				</div>
			</a>

			{#if showTrash}
				<a
					href={trashHref}
					onclick={closeUserMenu}
					class="flex items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
				>
					<Trash class="h-4 w-4" />
					<span>{m.topbar_trash()}</span>
				</a>
			{/if}

			<div class="my-1 h-px bg-zinc-200 dark:bg-zinc-700"></div>
			<button
				type="button"
				onclick={handleLogout}
				class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-red-600 hover:bg-zinc-100 dark:text-red-400 dark:hover:bg-zinc-700"
			>
				<SignOut class="h-4 w-4" />
				<span>{m.topbar_logout()}</span>
			</button>
		</div>
	{/if}
</div>
