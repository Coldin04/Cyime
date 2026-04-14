<script lang="ts">
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import { getLocale, isLocale, locales } from '$paraglide/runtime';
	import UserAvatar from '$lib/components/common/UserAvatar.svelte';
	import { auth } from '$lib/stores/auth';
	import { updateDisplayName } from '$lib/api/user';
	import AvatarEditDialog from '$lib/components/user/AvatarEditDialog.svelte';
	import {
		clearManualLocaleCookie,
		getManualLocaleFromDocument,
		setManualLocaleCookie
	} from '$lib/paraglide/manual-locale-cookie';

	let displayName = $state('');
	let avatarDialogOpen = $state(false);
	let savingDisplayName = $state(false);
	type LocalePreference = 'system' | (typeof locales)[number];
	let localePreference = $state<LocalePreference>('system');
	let switchingLocale = $state(false);

	$effect(() => {
		displayName = $auth.user?.displayName ?? '';
	});

	onMount(() => {
		const manualLocale = getManualLocaleFromDocument();
		localePreference = manualLocale && isLocale(manualLocale) ? manualLocale : 'system';
	});

	function getLocaleOptionLabel(localeTag: string): string {
		if (!browser || typeof Intl === 'undefined' || typeof Intl.DisplayNames === 'undefined') {
			return localeTag;
		}
		try {
			const display = new Intl.DisplayNames([getLocale()], { type: 'language' });
			return display.of(localeTag.split('-')[0]) ?? localeTag;
		} catch {
			return localeTag;
		}
	}

	async function handleDisplayNameSubmit(event?: SubmitEvent) {
		event?.preventDefault();
		const nextName = displayName.trim();
		if (!nextName) {
			toast.error(m.user_profile_display_name_required());
			return;
		}

		savingDisplayName = true;
		try {
			const user = await updateDisplayName(nextName);
			auth.setUser(user);
			toast.success(m.user_profile_display_name_updated());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_profile_display_name_update_failed());
		} finally {
			savingDisplayName = false;
		}
	}

	async function handleLocaleChange(event: Event) {
		const next = (event.currentTarget as HTMLSelectElement | null)?.value;
		if (!next) return;
		if (next !== 'system' && !isLocale(next)) return;
		if (next === localePreference) return;

		switchingLocale = true;
		try {
			if (next === 'system') {
				clearManualLocaleCookie();
				localePreference = 'system';
				if (browser) {
					window.location.reload();
				}
				return;
			}

			if (!isLocale(next)) return;
			setManualLocaleCookie(next);
			localePreference = next as LocalePreference;
			if (browser) {
				window.location.reload();
			}
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_profile_language_update_failed());
		} finally {
			switchingLocale = false;
		}
	}

</script>

<div class="flex flex-col gap-4">
	<!-- Avatar Row -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between border-b border-zinc-100 pb-6 dark:border-zinc-800/50">
		<div class="space-y-1 pr-4">
			<h2 class="text-base font-medium text-zinc-900 dark:text-zinc-100">{m.user_profile_avatar_title()}</h2>
			<p class="text-xs text-zinc-500 dark:text-zinc-400">{m.user_profile_avatar_description()}</p>
		</div>
		<button
			type="button"
			class="group relative shrink-0 rounded-full focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-zinc-900"
			onclick={() => (avatarDialogOpen = true)}
		>
			<UserAvatar size={64} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
			<span class="pointer-events-none absolute inset-0 rounded-full bg-black/0 transition group-hover:bg-black/10 dark:group-hover:bg-white/10"></span>
		</button>
	</div>

	<!-- Display Name Row -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between pt-6">
		<div class="space-y-1 sm:w-1/3">
			<h2 class="text-base font-medium text-zinc-900 dark:text-zinc-100">{m.user_profile_display_name_title()}</h2>
			<p class="text-xs text-zinc-500 dark:text-zinc-400">{m.user_profile_display_name_description()}</p>
		</div>
		<form class="flex w-full flex-1 gap-3 sm:max-w-md" onsubmit={handleDisplayNameSubmit}>
			<input
				bind:value={displayName}
				type="text"
				maxlength="80"
				class="w-full flex-1 rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-cyan-500 dark:focus:ring-cyan-900/60"
				placeholder={m.user_profile_display_name_placeholder()}
			/>
			<button
				type="submit"
				class="shrink-0 rounded-lg bg-zinc-900 px-5 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
				disabled={savingDisplayName}
			>
				{savingDisplayName ? m.common_saving() : m.common_save()}
			</button>
		</form>
	</div>

	<!-- Language Row -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between pt-4">
		<div class="space-y-1 sm:w-1/3">
			<h2 class="text-base font-medium text-zinc-900 dark:text-zinc-100">{m.user_profile_language_title()}</h2>
			<p class="text-xs text-zinc-500 dark:text-zinc-400">{m.user_profile_language_description()}</p>
		</div>
		<div class="flex w-full flex-1 gap-3 sm:max-w-md">
			<select
				value={localePreference}
				onchange={handleLocaleChange}
				disabled={switchingLocale}
				class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-200 disabled:cursor-not-allowed disabled:opacity-60 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-cyan-500 dark:focus:ring-cyan-900/60"
			>
				<option value="system">{m.user_profile_language_option_system()}</option>
				{#each locales as localeTag (localeTag)}
					<option value={localeTag}>{getLocaleOptionLabel(localeTag)}</option>
				{/each}
			</select>
		</div>
	</div>

	<!-- Email Row -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between pt-4">
		<div class="space-y-1 sm:w-1/3">
			<h2 class="text-base font-medium text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
				{m.user_profile_email_title()}
				<span class="rounded bg-zinc-100 px-1.5 py-0.5 text-[10px] font-medium text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">{m.user_profile_badge_wip()}</span>
			</h2>
			<p class="text-xs text-zinc-500 dark:text-zinc-400">{m.user_profile_email_description()}</p>
		</div>
		<div class="flex w-full flex-1 gap-3 sm:max-w-md">
			<input
				value={$auth.user?.email || m.user_common_no_email()}
				type="text"
				disabled
				class="w-full flex-1 rounded-lg border border-zinc-200 bg-zinc-50/50 px-4 py-2 text-sm text-zinc-500 outline-none cursor-not-allowed dark:border-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-500"
			/>
			<button
				type="button"
				disabled
				class="shrink-0 rounded-lg border border-zinc-200 bg-transparent px-5 py-2 text-sm font-medium text-zinc-400 cursor-not-allowed dark:border-zinc-800 dark:text-zinc-600"
			>
				{m.user_profile_email_action_edit()}
			</button>
		</div>
	</div>
</div>

<AvatarEditDialog bind:open={avatarDialogOpen} />
