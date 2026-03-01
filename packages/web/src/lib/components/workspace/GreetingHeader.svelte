<script lang="ts">
	import { auth } from '$lib/stores/auth';

	function getGreeting(): string {
		const hour = new Date().getHours();
		if (hour < 6) {
			return '夜深了';
		} else if (hour < 12) {
			return '早上好';
		} else if (hour < 14) {
			return '中午好';
		} else if (hour < 18) {
			return '下午好';
		} else {
			return '晚上好';
		}
	}

	function getInitial(name: string | null): string {
		if (!name || name.trim() === '') {
			return 'U';
		}
		return name.charAt(0).toUpperCase();
	}
</script>

<section class="my-8 flex items-center gap-4">
	<div
		class="grid h-16 w-16 flex-shrink-0 place-content-center rounded-full bg-riptide-100 dark:bg-riptide-900"
	>
		<span class="text-3xl font-bold text-riptide-600 dark:text-riptide-300">
			{getInitial($auth.user?.displayName || null)}
		</span>
	</div>
	<div>
		<h2 class="text-2xl font-bold text-zinc-800 dark:text-zinc-200">
			{getGreeting()}, {$auth.user?.displayName || 'User'}
		</h2>
		<p class="text-zinc-500 dark:text-zinc-400">今天有什么新灵感？</p>
	</div>
</section>
