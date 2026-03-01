<script lang="ts">
	import MagnifyingGlass from '~icons/ph/magnifying-glass';
	import User from '~icons/ph/user';
</script>

<nav
	class="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-black/10 bg-white/80 px-4 backdrop-blur-md dark:border-white/10 dark:bg-zinc-900/80"
>
	<div class="flex items-center gap-2">
		<span class="bg-gradient-to-r from-cyan-300 to-yellow-300 bg-clip-text font-bold text-transparent text-lg">CyimeWrite</span>
	</div>

	<div class="flex items-center gap-4">
		<button
			class="grid h-8 w-8 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
		>
			<MagnifyingGlass class="h-5 w-5" />
		</button>
		<button
			class="grid h-8 w-8 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
		>
			<User class="h-5 w-5" />
		</button>
	</div>
</nav>
