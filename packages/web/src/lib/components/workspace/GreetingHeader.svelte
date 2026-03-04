<script lang="ts">
	import { auth } from '$lib/stores/auth';
	import * as m from '$paraglide/messages';

	function getGreeting(): string {
		const hour = new Date().getHours();
		if (hour < 6) {
			return m.greeting_night();
		} else if (hour < 12) {
			return m.greeting_morning();
		} else if (hour < 14) {
			return m.greeting_noon();
		} else if (hour < 18) {
			return m.greeting_afternoon();
		} else {
			return m.greeting_evening();
		}
	}

	function getInitial(name: string | null): string {
		if (!name || name.trim() === '') {
			return 'U';
		}
		return name.charAt(0).toUpperCase();
	}
</script>

<section class="mb-6 flex items-center gap-4">
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
		<p class="text-zinc-500 dark:text-zinc-400">{m.greeting_question()}</p>
	</div>
</section>
