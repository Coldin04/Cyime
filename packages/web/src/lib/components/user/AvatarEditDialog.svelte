<script lang="ts">
	import { browser } from '$app/environment';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import { portal } from '$lib/actions/portal';
	import UserAvatar from '$lib/components/common/UserAvatar.svelte';
	import { avatarMaxBytes, avatarOutputSize } from '$lib/config/avatar';
	import { auth } from '$lib/stores/auth';
	import { setGitHubAvatar, uploadAvatar } from '$lib/api/user';
	import Cropper from 'svelte-easy-crop';
	import { getCroppedImg } from '$lib/utils/image';

	let { open = $bindable(false) } = $props<{ open: boolean }>();

	let githubUsername = $state('');
	let uploadingAvatar = $state(false);
	let savingGitHubAvatar = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

	let cropperPreviewUrl = $state('');
	let crop = $state({ x: 0, y: 0 });
	let zoom = $state(1);
	let pixelCrop = $state<{ x: number; y: number; width: number; height: number } | null>(null);

	function closeDialog() {
		open = false;
	}

	async function handleAvatarFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		try {
			if (!file.type.startsWith('image/')) {
				toast.error(m.user_avatar_error_invalid_type());
				return;
			}
			openCropper(file);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_avatar_error_upload_failed());
		} finally {
			if (input) {
				input.value = '';
			}
		}
	}

	async function handleGitHubAvatarSubmit(event?: SubmitEvent) {
		event?.preventDefault();
		const username = githubUsername.trim();
		if (!username) {
			toast.error(m.user_avatar_error_github_empty());
			return;
		}

		savingGitHubAvatar = true;
		try {
			const user = await setGitHubAvatar(username);
			auth.setUser(user);
			githubUsername = '';
			open = false;
			toast.success(m.user_avatar_success_github());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_avatar_error_github_failed());
		} finally {
			savingGitHubAvatar = false;
		}
	}

	function openCropper(file: File) {
		if (!browser) {
			throw new Error('Not in browser environment');
		}
		clearCropper();
		cropperPreviewUrl = URL.createObjectURL(file);
		crop = { x: 0, y: 0 };
		zoom = 1;
		pixelCrop = null;
	}

	async function handleCroppedUpload() {
		if (!pixelCrop || !cropperPreviewUrl) {
			toast.error(m.user_avatar_error_no_image());
			return;
		}
		uploadingAvatar = true;
		try {
			const blob = await getCroppedImg(cropperPreviewUrl, pixelCrop, avatarOutputSize);
			const prepared = new File([blob], 'avatar.jpg', { type: 'image/jpeg' });
			if (prepared.size > avatarMaxBytes) {
				const maxMB = (avatarMaxBytes / (1024 * 1024)).toFixed(1).replace(/\.0$/, '');
				toast.error(m.user_avatar_error_too_large({ size: maxMB }));
				return;
			}

			const user = await uploadAvatar(prepared);
			auth.setUser(user);
			toast.success(m.user_avatar_success_updated());
			clearCropper();
			open = false;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_avatar_error_upload_failed());
		} finally {
			uploadingAvatar = false;
		}
	}

	function clearCropper() {
		if (cropperPreviewUrl) {
			URL.revokeObjectURL(cropperPreviewUrl);
			cropperPreviewUrl = '';
		}
		pixelCrop = null;
	}

	$effect(() => {
		if (browser) {
			if (open) {
				document.body.style.overflow = 'hidden';
			} else {
				document.body.style.overflow = '';
			}
		}

		if (!open) {
			clearCropper();
		}

		return () => {
			if (browser) {
				document.body.style.overflow = '';
			}
		};
	});
</script>

{#if open}
	<div
		use:portal
		class="fixed inset-0 z-[100] min-h-dvh w-screen overflow-y-auto bg-black/40"
		role="presentation"
		onclick={closeDialog}
	>
		<div class="flex min-h-dvh w-full items-center justify-center p-2 sm:p-4">
			<div
				class={`w-full border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-950 max-w-md rounded-3xl p-5`}
				role="dialog"
				aria-modal="true"
				tabindex="-1"
				onclick={(event) => event.stopPropagation()}
				onkeydown={(event) => {
					if (event.key === 'Escape') {
						closeDialog();
					}
				}}
			>
				<div class="flex items-start justify-between gap-4">
					<div>
						<h2 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">{m.user_avatar_dialog_title()}</h2>
						<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">{m.user_avatar_dialog_description()}</p>
					</div>
					<button
						type="button"
						class="rounded-full p-2 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
						onclick={closeDialog}
					>
						✕
					</button>
				</div>

				{#if cropperPreviewUrl}
					<div class="mt-5 space-y-4">
						<div class="relative h-64 w-full overflow-hidden rounded-2xl bg-zinc-100 dark:bg-zinc-900">
							<Cropper
								image={cropperPreviewUrl}
								bind:crop
								bind:zoom
								aspect={1}
								cropShape="round"
								oncropcomplete={(e) => (pixelCrop = e.pixels)}
							/>
						</div>

						<div class="flex flex-col gap-2 pt-2">
							<div class="flex items-center gap-3 px-2">
								<span class="text-xs font-medium text-zinc-500 dark:text-zinc-400">{m.user_avatar_zoom()}</span>
								<input
									type="range"
									min="1"
									max="3"
									step="0.1"
									bind:value={zoom}
									class="h-1.5 flex-1 cursor-pointer appearance-none rounded-full bg-zinc-200 dark:bg-zinc-700 [&::-webkit-slider-thumb]:h-4 [&::-webkit-slider-thumb]:w-4 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-zinc-900 dark:[&::-webkit-slider-thumb]:bg-zinc-100"
								/>
							</div>
						</div>

						<div class="flex gap-3 pt-2">
							<button
								type="button"
								class="flex-1 rounded-xl border border-zinc-200 px-4 py-2.5 text-sm font-medium text-zinc-700 transition hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
								onclick={clearCropper}
								disabled={uploadingAvatar}
							>
								{m.common_back()}
							</button>
							<button
								type="button"
								class="flex-[2] rounded-xl bg-zinc-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
								onclick={handleCroppedUpload}
								disabled={uploadingAvatar}
							>
								{uploadingAvatar ? m.common_uploading() : m.user_avatar_confirm_upload()}
							</button>
						</div>
					</div>
				{:else}
					<div class="mt-5 flex items-center justify-between gap-4">
						<div class="flex min-w-0 items-center gap-4">
							<UserAvatar size={64} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
							<div class="min-w-0">
								<p class="truncate text-sm font-semibold text-zinc-900 dark:text-zinc-100">
									{$auth.user?.displayName || m.user_common_default_name()}
								</p>
								<p class="truncate text-xs text-zinc-500 dark:text-zinc-400">
									{$auth.user?.email || m.user_common_no_email()}
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
							{m.user_avatar_select_image()}
						</button>
					</div>

					<div class="mt-5 space-y-4">
						<form class="space-y-2" onsubmit={handleGitHubAvatarSubmit}>
							<p class="text-sm font-medium text-zinc-900 dark:text-zinc-100">{m.user_avatar_github_username()}</p>
							<div class="flex gap-2">
								<input
									bind:value={githubUsername}
									type="text"
									autocapitalize="off"
									autocomplete="off"
									spellcheck="false"
									class="min-w-0 flex-1 rounded-xl border border-zinc-200 bg-white px-4 py-3 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
									placeholder={m.user_avatar_github_placeholder()}
								/>
								<button
									type="submit"
									class="rounded-xl bg-zinc-900 px-4 py-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
									disabled={savingGitHubAvatar}
								>
									{savingGitHubAvatar ? m.common_saving() : m.common_apply()}
								</button>
							</div>
						</form>
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}
