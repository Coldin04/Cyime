<script lang="ts">
	// Placeholder for user data
	const user = {
		name: 'User',
		initial: 'U'
	};
</script>

<section class="my-8 flex items-center gap-4">
	<div
		class="grid h-16 w-16 flex-shrink-0 place-content-center rounded-full bg-riptide-100 dark:bg-riptide-900"
	>
		<span class="text-3xl font-bold text-riptide-600 dark:text-riptide-300">{user.initial}</span>
	</div>
	<div>
		<h2 class="text-2xl font-bold text-zinc-800 dark:text-zinc-200">
			早上好, {user.name}
		</h2>
		<p class="text-zinc-500 dark:text-zinc-400">今天有什么新灵感？</p>
	</div>
</section>
