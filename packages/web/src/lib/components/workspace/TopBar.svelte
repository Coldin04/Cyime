<script lang="ts">
	import MagnifyingGlass from '~icons/ph/magnifying-glass';
	import User from '~icons/ph/user';
	import SignOut from '~icons/ph/sign-out';
	import Trash from '~icons/ph/trash';
	import { auth } from '$lib/stores/auth';

	let showUserMenu = $state(false);

	function toggleUserMenu() {
		showUserMenu = !showUserMenu;
	}

	function handleLogout() {
		auth.logout();
		showUserMenu = false;
	}
</script>

<nav
	class="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-black/10 bg-white/80 px-4 backdrop-blur-md dark:border-white/10 dark:bg-zinc-900/80"
>
	<div class="flex items-center gap-2">
		<a href="/workspace" class="bg-gradient-to-r from-cyan-300 to-yellow-300 bg-clip-text font-bold text-transparent text-lg">CyimeWrite</a>
	</div>

	<div class="flex items-center gap-4">
		<button
			class="grid h-8 w-8 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
		>
			<MagnifyingGlass class="h-5 w-5" />
		</button>
		<div class="relative">
			<button
				onclick={toggleUserMenu}
				class="grid h-8 w-8 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
			>
				<User class="h-5 w-5" />
			</button>
			{#if showUserMenu}
				<div
					class="absolute top-full right-0 z-10 mt-2 w-48 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-zinc-800 dark:ring-zinc-700"
				>
					<a
						href="#"
						class="block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
						>个人资料</a
					>
					<a
						href="/workspace/trash"
						class="flex items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
					>
						<Trash class="h-4 w-4" />
						<span>回收站</span>
					</a>
					<div class="my-1 h-px bg-zinc-200 dark:bg-zinc-700"></div>
					<button
						onclick={handleLogout}
						class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-red-600 hover:bg-zinc-100 dark:text-red-400 dark:hover:bg-zinc-700"
					>
						<SignOut class="h-4 w-4" />
						<span>登出</span>
					</button>
				</div>
			{/if}
		</div>
	</div>
</nav>
