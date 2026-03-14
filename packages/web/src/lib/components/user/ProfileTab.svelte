<script lang="ts">
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import UserAvatar from '$lib/components/common/UserAvatar.svelte';
	import { auth } from '$lib/stores/auth';
	import { updateDisplayName } from '$lib/api/user';
	import AvatarEditDialog from '$lib/components/user/AvatarEditDialog.svelte';

	let displayName = $state('');
	let avatarDialogOpen = $state(false);
	let savingDisplayName = $state(false);

	$effect(() => {
		displayName = $auth.user?.displayName ?? '';
	});

	async function handleDisplayNameSubmit(event?: SubmitEvent) {
		event?.preventDefault();
		const nextName = displayName.trim();
		if (!nextName) {
			toast.error(m.user_avatar_error_invalid_type()); // Generic error for now or add a new one
			return;
		}

		savingDisplayName = true;
		try {
			const user = await updateDisplayName(nextName);
			auth.setUser(user);
			toast.success(m.user_avatar_success_updated());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Update failed');
		} finally {
			savingDisplayName = false;
		}
	}
</script>

<div class="space-y-8">
	<!-- Avatar Row -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between border-b border-zinc-100 pb-8 dark:border-zinc-800/50">
		<div class="space-y-1 pr-4">
			<h2 class="text-base font-medium text-zinc-900 dark:text-zinc-100">头像</h2>
			<p class="text-sm text-zinc-500 dark:text-zinc-400">点击头像可修改或上传新图片。</p>
		</div>
		<button
			type="button"
			class="group relative shrink-0 rounded-full focus:outline-none focus:ring-2 focus:ring-riptide-500 focus:ring-offset-2 dark:focus:ring-offset-zinc-900"
			onclick={() => (avatarDialogOpen = true)}
		>
			<UserAvatar size={72} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
			<span class="pointer-events-none absolute inset-0 rounded-full bg-black/0 transition group-hover:bg-black/10 dark:group-hover:bg-white/10"></span>
		</button>
	</div>

	<!-- Display Name Row -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between border-b border-zinc-100 pb-8 dark:border-zinc-800/50">
		<div class="space-y-1 sm:w-1/3">
			<h2 class="text-base font-medium text-zinc-900 dark:text-zinc-100">昵称</h2>
			<p class="text-sm text-zinc-500 dark:text-zinc-400">用于在平台中展示的名称。</p>
		</div>
		<form class="flex w-full flex-1 gap-3 sm:max-w-md" onsubmit={handleDisplayNameSubmit}>
			<input
				bind:value={displayName}
				type="text"
				maxlength="80"
				class="w-full flex-1 rounded-xl border border-zinc-200 bg-transparent px-4 py-2.5 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
				placeholder="输入你想展示的昵称"
			/>
			<button
				type="submit"
				class="shrink-0 rounded-xl bg-zinc-900 px-5 py-2.5 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
				disabled={savingDisplayName}
			>
				{savingDisplayName ? m.common_saving() : m.common_save()}
			</button>
		</form>
	</div>

	<!-- Email Row -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div class="space-y-1 sm:w-1/3">
			<h2 class="text-base font-medium text-zinc-900 dark:text-zinc-100">邮箱</h2>
			<p class="text-sm text-zinc-500 dark:text-zinc-400">关联和登录使用的邮箱地址。</p>
		</div>
		<div class="flex w-full flex-1 gap-3 sm:max-w-md">
			<input
				value={$auth.user?.email || 'No email'}
				type="text"
				disabled
				class="w-full flex-1 rounded-xl border border-zinc-200 bg-zinc-50/50 px-4 py-2.5 text-sm text-zinc-500 outline-none cursor-not-allowed dark:border-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-500"
			/>
			<button
				type="button"
				disabled
				class="shrink-0 rounded-xl border border-zinc-200 bg-transparent px-5 py-2.5 text-sm font-medium text-zinc-400 cursor-not-allowed dark:border-zinc-800 dark:text-zinc-600"
			>
				修改
			</button>
		</div>
	</div>
</div>

<AvatarEditDialog bind:open={avatarDialogOpen} />
