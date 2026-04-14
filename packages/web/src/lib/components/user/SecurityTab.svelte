<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import { listAuthSessions, revokeAuthSession, revokeOtherAuthSessions, type AuthSession } from '$lib/api/auth';
	import { auth } from '$lib/stores/auth';

	let sessions = $state<AuthSession[]>([]);
	let loading = $state(true);
	let revokingSessionId = $state('');
	let revokingOthers = $state(false);

	const currentSession = $derived(sessions.find((session) => session.current) ?? null);
	const otherSessions = $derived(sessions.filter((session) => !session.current));

	onMount(() => {
		void loadSessions();
	});

	function formatDateTime(value: string): string {
		const date = new Date(value);
		if (Number.isNaN(date.getTime())) return value;
		return date.toLocaleString();
	}

	async function loadSessions() {
		loading = true;
		try {
			sessions = await listAuthSessions();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_security_sessions_load_failed());
		} finally {
			loading = false;
		}
	}

	async function handleRevokeSession(session: AuthSession) {
		revokingSessionId = session.id;
		try {
			await revokeAuthSession(session.id);
			if (session.current) {
				await auth.logout();
				return;
			}
			sessions = sessions.filter((item) => item.id !== session.id);
			toast.success(m.user_security_session_revoked());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_security_session_revoke_failed());
		} finally {
			revokingSessionId = '';
		}
	}

	async function handleRevokeOthers() {
		revokingOthers = true;
		try {
			const revokedCount = await revokeOtherAuthSessions();
			sessions = sessions.filter((session) => session.current);
			toast.success(m.user_security_other_sessions_revoked({ count: revokedCount }));
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_security_session_revoke_failed());
		} finally {
			revokingOthers = false;
		}
	}
</script>

<div class="space-y-6">
	<div class="space-y-5">
		<div class="flex items-start justify-between gap-4">
			<div class="space-y-1">
				<h2 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{m.user_security_sessions_title()}</h2>
				<p class="text-sm text-zinc-500 dark:text-zinc-400">{m.user_security_sessions_description()}</p>
			</div>
			{#if otherSessions.length > 0}
				<button
					type="button"
					class="hidden shrink-0 rounded-lg border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800 sm:inline-flex"
					disabled={revokingOthers}
					onclick={handleRevokeOthers}
				>
					{revokingOthers ? m.common_loading() : m.user_security_revoke_other_sessions()}
				</button>
			{/if}
		</div>

		<div class="space-y-4">
			{#if loading}
				<div class="rounded-xl border border-dashed border-zinc-200 px-4 py-6 text-sm text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
					{m.common_loading()}
				</div>
			{:else if sessions.length === 0}
				<div class="rounded-xl border border-dashed border-zinc-200 px-4 py-6 text-sm text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
					{m.user_security_sessions_empty()}
				</div>
			{:else}
				{#if currentSession}
					<section class="rounded-xl border border-cyan-200/80 bg-cyan-50/50 p-4 dark:border-cyan-900/50 dark:bg-cyan-950/20">
						<div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
							<div class="min-w-0 flex-1 space-y-3">
								<div class="flex flex-wrap items-center gap-2">
									<h3 class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{currentSession.deviceLabel || m.user_security_device_unknown()}</h3>
									<span class="rounded-full bg-cyan-100 px-2 py-0.5 text-[11px] font-medium text-cyan-800 dark:bg-cyan-900/50 dark:text-cyan-200">
										{m.user_security_current_session_badge()}
									</span>
								</div>
								<p
									class="max-w-full overflow-hidden text-ellipsis whitespace-nowrap text-xs text-zinc-500 dark:text-zinc-400"
									title={currentSession.userAgent || m.user_security_device_unknown()}
								>
									{currentSession.userAgent || m.user_security_device_unknown()}
								</p>

								<div class="grid gap-3 text-xs text-zinc-600 dark:text-zinc-300 sm:grid-cols-2">
									<div>
										<p class="font-medium text-zinc-900 dark:text-zinc-100">{m.user_security_last_seen_label()}</p>
										<p>{formatDateTime(currentSession.lastSeenAt)}</p>
									</div>
									<div>
										<p class="font-medium text-zinc-900 dark:text-zinc-100">{m.user_security_expires_at_label()}</p>
										<p>{formatDateTime(currentSession.expiresAt)}</p>
									</div>
								</div>
							</div>
							<button
								type="button"
								class="w-full shrink-0 rounded-lg bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200 sm:w-auto"
								disabled={revokingSessionId === currentSession.id}
								onclick={() => handleRevokeSession(currentSession)}
							>
								{revokingSessionId === currentSession.id ? m.common_loading() : m.user_security_sign_out_current()}
							</button>
						</div>
					</section>
				{/if}

				<section class="space-y-3">
					<div class="flex items-center justify-between gap-3">
						<h3 class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{m.user_security_other_sessions_title()}</h3>
						<p class="hidden text-xs text-zinc-500 dark:text-zinc-400 sm:block">{m.user_security_other_sessions_hint()}</p>
					</div>

					{#if otherSessions.length > 0}
						<button
							type="button"
							class="w-full rounded-lg border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800 sm:hidden"
							disabled={revokingOthers}
							onclick={handleRevokeOthers}
						>
							{revokingOthers ? m.common_loading() : m.user_security_revoke_other_sessions()}
						</button>
					{/if}

					{#if otherSessions.length === 0}
						<div class="rounded-xl border border-dashed border-zinc-200 px-4 py-6 text-sm text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
							{m.user_security_other_sessions_empty()}
						</div>
					{:else}
						<div class="space-y-3">
							{#each otherSessions as session (session.id)}
								<article class="rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
									<div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
										<div class="min-w-0 flex-1 space-y-3">
											<h4 class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{session.deviceLabel || m.user_security_device_unknown()}</h4>
											<p
												class="max-w-full overflow-hidden text-ellipsis whitespace-nowrap text-xs text-zinc-500 dark:text-zinc-400"
												title={session.userAgent || m.user_security_device_unknown()}
											>
												{session.userAgent || m.user_security_device_unknown()}
											</p>

											<div class="grid gap-3 text-xs text-zinc-600 dark:text-zinc-300 sm:grid-cols-2">
												<div>
													<p class="font-medium text-zinc-900 dark:text-zinc-100">{m.user_security_last_seen_label()}</p>
													<p>{formatDateTime(session.lastSeenAt)}</p>
												</div>
												<div>
													<p class="font-medium text-zinc-900 dark:text-zinc-100">{m.user_security_expires_at_label()}</p>
													<p>{formatDateTime(session.expiresAt)}</p>
												</div>
											</div>
										</div>
										<button
											type="button"
											class="w-full shrink-0 rounded-lg border border-rose-200 px-3 py-2 text-sm font-medium text-rose-600 transition hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-900/50 dark:text-rose-300 dark:hover:bg-rose-950/20 sm:w-auto"
											disabled={revokingSessionId === session.id}
											onclick={() => handleRevokeSession(session)}
										>
											{revokingSessionId === session.id ? m.common_loading() : m.user_security_revoke_session_action()}
										</button>
									</div>
								</article>
							{/each}
						</div>
					{/if}
				</section>
			{/if}
		</div>
	</div>

	<div class="rounded-2xl border border-zinc-200 bg-zinc-50 p-5 text-sm text-zinc-500 dark:border-zinc-700/50 dark:bg-zinc-800/20 dark:text-zinc-400">
		<p class="mb-1 font-medium text-zinc-900 dark:text-zinc-100">{m.user_security_next_step_title()}</p>
		<p>{m.user_security_next_step_description()}</p>
	</div>
</div>
