<script lang="ts">
	import * as m from '$paraglide/messages';
	import { toast } from 'svelte-sonner';
	import UserAvatar from '$lib/components/common/UserAvatar.svelte';
	import { avatarMaxBytes, avatarOutputSize } from '$lib/config/avatar';
	import { auth } from '$lib/stores/auth';
	import { setGitHubAvatar, updateDisplayName, uploadAvatar } from '$lib/api/user';

	let displayName = $state('');
	let githubUsername = $state('');
	let avatarDialogOpen = $state(false);
	let savingDisplayName = $state(false);
	let uploadingAvatar = $state(false);
	let savingGitHubAvatar = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

	$effect(() => {
		displayName = $auth.user?.displayName ?? '';
	});

	async function handleDisplayNameSubmit(event?: SubmitEvent) {
		event?.preventDefault();
		const nextName = displayName.trim();
		if (!nextName) {
			toast.error('昵称不能为空');
			return;
		}

		savingDisplayName = true;
		try {
			const user = await updateDisplayName(nextName);
			auth.setUser(user);
			toast.success('昵称已更新');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '昵称更新失败');
		} finally {
			savingDisplayName = false;
		}
	}

	async function handleAvatarFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		uploadingAvatar = true;
		try {
			const prepared = await prepareAvatarFile(file);
			if (prepared.size > avatarMaxBytes) {
				const maxMB = (avatarMaxBytes / (1024 * 1024)).toFixed(1).replace(/\.0$/, '');
				toast.error(`头像处理后仍超过 ${maxMB}MB，请换一张更小的图片`);
				return;
			}

			const user = await uploadAvatar(prepared);
			auth.setUser(user);
			avatarDialogOpen = false;
			toast.success('头像已更新');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '头像上传失败');
		} finally {
			uploadingAvatar = false;
			if (input) {
				input.value = '';
			}
		}
	}

	async function handleGitHubAvatarSubmit(event?: SubmitEvent) {
		event?.preventDefault();
		const username = githubUsername.trim();
		if (!username) {
			toast.error('GitHub 用户名不能为空');
			return;
		}

		savingGitHubAvatar = true;
		try {
			const user = await setGitHubAvatar(username);
			auth.setUser(user);
			githubUsername = '';
			avatarDialogOpen = false;
			toast.success('GitHub 头像已更新');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'GitHub 头像更新失败');
		} finally {
			savingGitHubAvatar = false;
		}
	}

	async function prepareAvatarFile(file: File): Promise<File> {
		if (!file.type.startsWith('image/')) {
			throw new Error('请选择图片文件');
		}

		const img = await loadImage(file);
		const sourceSize = Math.min(img.width, img.height);
		const sourceX = Math.floor((img.width - sourceSize) / 2);
		const sourceY = Math.floor((img.height - sourceSize) / 2);

		const canvas = document.createElement('canvas');
		canvas.width = avatarOutputSize;
		canvas.height = avatarOutputSize;
		const ctx = canvas.getContext('2d');
		if (!ctx) {
			throw new Error('无法处理头像图片');
		}

		ctx.imageSmoothingEnabled = true;
		ctx.imageSmoothingQuality = 'high';
		ctx.drawImage(
			img,
			sourceX,
			sourceY,
			sourceSize,
			sourceSize,
			0,
			0,
			avatarOutputSize,
			avatarOutputSize
		);

		const blob = await canvasToBlob(canvas, 'image/jpeg', 0.9);
		return new File([blob], 'avatar.jpg', { type: 'image/jpeg' });
	}

	function loadImage(file: File): Promise<HTMLImageElement> {
		return new Promise((resolve, reject) => {
			const objectURL = URL.createObjectURL(file);
			const img = new Image();
			img.onload = () => {
				URL.revokeObjectURL(objectURL);
				resolve(img);
			};
			img.onerror = () => {
				URL.revokeObjectURL(objectURL);
				reject(new Error('图片加载失败'));
			};
			img.src = objectURL;
		});
	}

	function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> {
		return new Promise((resolve, reject) => {
			canvas.toBlob(
				(blob) => {
					if (!blob) {
						reject(new Error('头像处理失败'));
						return;
					}
					resolve(blob);
				},
				type,
				quality
			);
		});
	}
</script>

<svelte:head>
	<title>{m.page_title_user_profile()}</title>
</svelte:head>

<section class="space-y-4">
	<h1 class="text-2xl font-bold text-zinc-900 dark:text-zinc-100">{m.user_profile_title()}</h1>
	<p class="text-sm text-zinc-600 dark:text-zinc-400">
		{m.user_profile_description()}
	</p>
	<section class="space-y-6 rounded-2xl border border-zinc-200 bg-zinc-50/70 p-5 dark:border-zinc-700 dark:bg-zinc-800/40">
		<div class="flex flex-col items-start gap-4 sm:flex-row sm:items-center">
			<button
				type="button"
				class="group relative rounded-full"
				onclick={() => (avatarDialogOpen = true)}
			>
				<UserAvatar size={88} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
				<span class="pointer-events-none absolute inset-0 rounded-full bg-black/0 transition group-hover:bg-black/10 dark:group-hover:bg-white/10"></span>
			</button>
			<div class="min-w-0">
				<p class="truncate text-lg font-semibold text-zinc-900 dark:text-zinc-100">
					{$auth.user?.displayName || 'User'}
				</p>
				<p class="truncate text-sm text-zinc-500 dark:text-zinc-400">{$auth.user?.email || 'No email'}</p>
				<button
					type="button"
					class="mt-3 text-sm font-medium text-riptide-700 transition hover:text-riptide-800 dark:text-riptide-300 dark:hover:text-riptide-200"
					onclick={() => (avatarDialogOpen = true)}
				>
					修改头像
				</button>
			</div>
		</div>

		<section class="max-w-xl space-y-3 rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-900">
			<div class="space-y-1">
				<h2 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">昵称</h2>
				<p class="text-sm text-zinc-500 dark:text-zinc-400">只修改显示名称。</p>
			</div>
			<form class="space-y-3" onsubmit={handleDisplayNameSubmit}>
				<input
					bind:value={displayName}
					type="text"
					maxlength="80"
					class="w-full rounded-xl border border-zinc-200 bg-white px-4 py-3 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
					placeholder="输入你想展示的昵称"
				/>
				<div class="flex justify-end">
					<button
						type="submit"
						class="rounded-xl bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
						disabled={savingDisplayName}
					>
						{savingDisplayName ? '保存中...' : '保存昵称'}
					</button>
				</div>
			</form>
		</section>
	</section>
</section>

{#if avatarDialogOpen}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4" role="presentation">
		<div class="w-full max-w-md rounded-3xl border border-zinc-200 bg-white p-5 shadow-2xl dark:border-zinc-800 dark:bg-zinc-950">
			<div class="flex items-start justify-between gap-4">
				<div>
					<h2 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">修改头像</h2>
					<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">上传一张图片，或直接使用 GitHub 头像。</p>
				</div>
				<button
					type="button"
					class="rounded-full p-2 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
					onclick={() => (avatarDialogOpen = false)}
				>
					✕
				</button>
			</div>

			<div class="mt-5 flex items-center justify-between gap-4">
				<div class="flex min-w-0 items-center gap-4">
					<UserAvatar size={64} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
					<div class="min-w-0">
						<p class="truncate text-sm font-semibold text-zinc-900 dark:text-zinc-100">
							{$auth.user?.displayName || 'User'}
						</p>
						<p class="truncate text-xs text-zinc-500 dark:text-zinc-400">
							{$auth.user?.email || 'No email'}
						</p>
					</div>
				</div>
				<input
					bind:this={fileInput}
					type="file"
					accept="image/png,image/jpeg,image/webp,image/gif"
					class="hidden"
					onchange={handleAvatarFileChange}
				/>
				<button
					type="button"
					class="shrink-0 rounded-xl bg-zinc-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
					disabled={uploadingAvatar}
					onclick={() => fileInput?.click()}
				>
					{uploadingAvatar ? '上传中...' : '选择图片'}
				</button>
			</div>

			<div class="mt-5 space-y-4">

				<form class="space-y-2" onsubmit={handleGitHubAvatarSubmit}>
					<p class="text-sm font-medium text-zinc-900 dark:text-zinc-100">GitHub 用户名</p>
					<div class="flex gap-2">
						<input
							bind:value={githubUsername}
							type="text"
							autocapitalize="off"
							autocomplete="off"
							spellcheck="false"
							class="min-w-0 flex-1 rounded-xl border border-zinc-200 bg-white px-4 py-3 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
							placeholder="例如 octocat"
						/>
						<button
							type="submit"
							class="rounded-xl bg-zinc-900 px-4 py-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
							disabled={savingGitHubAvatar}
						>
							{savingGitHubAvatar ? '保存中...' : '应用'}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
{/if}
