<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import {
		createImageBedConfig,
		deleteImageBedConfig,
		getImageBedConfigs,
		updateImageBedConfig,
		type ImageBedConfig,
		type UpsertImageBedConfigRequest
	} from '$lib/api/user';

	type ProviderType = 'see' | 'lsky';

	type FormState = {
		name: string;
		providerType: ProviderType;
		baseUrl: string;
		apiToken: string;
		isEnabled: boolean;
		storageId: number | null;
		strategyId: string;
	};

	const emptyForm = (): FormState => ({
		name: '',
		providerType: 'see',
		baseUrl: '',
		apiToken: '',
		isEnabled: true,
		storageId: null,
		strategyId: ''
	});

	let loading = $state(false);
	let saving = $state(false);
	let deletingId = $state<string | null>(null);
	let editingId = $state<string | null>(null);
	let items = $state<ImageBedConfig[]>([]);
	let form = $state<FormState>(emptyForm());

	onMount(() => {
		void loadConfigs();
	});

	async function loadConfigs() {
		loading = true;
		try {
			items = await getImageBedConfigs();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_image_beds_load_failed());
		} finally {
			loading = false;
		}
	}

	function startCreate() {
		editingId = null;
		form = emptyForm();
	}

	function startEdit(item: ImageBedConfig) {
		editingId = item.id;
		form = {
			name: item.name,
			providerType: item.providerType === 'lsky' ? 'lsky' : 'see',
			baseUrl: item.baseUrl ?? '',
			apiToken: item.apiToken ?? '',
			isEnabled: item.isEnabled,
			storageId: item.storageId ?? null,
			strategyId: item.strategyId ?? ''
		};
	}

	function toRequestBody(input: FormState): UpsertImageBedConfigRequest {
		return {
			name: input.name,
			providerType: input.providerType,
			baseUrl: input.providerType === 'lsky' ? input.baseUrl : '',
			apiToken: input.apiToken,
			isEnabled: input.isEnabled,
			storageId: input.providerType === 'lsky' ? (input.storageId ?? 0) : 0,
			strategyId: input.providerType === 'lsky' ? input.strategyId : ''
		};
	}

	async function handleSubmit(event?: SubmitEvent) {
		event?.preventDefault();
		saving = true;
		try {
			const request = toRequestBody(form);
			if (editingId) {
				const updated = await updateImageBedConfig(editingId, request);
				items = items.map((item) => (item.id === updated.id ? updated : item));
				toast.success(m.user_image_beds_updated());
			} else {
				const created = await createImageBedConfig(request);
				items = [...items, created];
				toast.success(m.user_image_beds_created());
			}
			startCreate();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_image_beds_update_failed());
		} finally {
			saving = false;
		}
	}

	async function handleDelete(id: string) {
		if (!window.confirm(m.user_image_beds_confirm_delete())) {
			return;
		}

		deletingId = id;
		try {
			await deleteImageBedConfig(id);
			items = items.filter((item) => item.id !== id);
			if (editingId === id) {
				startCreate();
			}
			toast.success(m.user_image_beds_deleted());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_image_beds_delete_failed());
		} finally {
			deletingId = null;
		}
	}
</script>

<div class="space-y-6">
	<div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950">
		<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
			<div class="space-y-1">
				<h2 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
					{m.user_image_beds_title()}
				</h2>
				<p class="text-sm text-zinc-500 dark:text-zinc-400">
					{m.user_image_beds_description()}
				</p>
			</div>
			<button
				type="button"
				class="shrink-0 rounded-lg border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-900"
				onclick={startCreate}
			>
				{m.user_image_beds_add()}
			</button>
		</div>

		<div class="mt-5 space-y-3">
			{#if loading}
				<p class="text-sm text-zinc-500 dark:text-zinc-400">{m.workspace_loading()}</p>
			{:else if items.length === 0}
				<div class="rounded-2xl border border-dashed border-zinc-300 px-4 py-6 text-sm text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
					{m.user_image_beds_empty()}
				</div>
			{:else}
				{#each items as item (item.id)}
					<div class="rounded-2xl border border-zinc-200 p-4 dark:border-zinc-800">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
							<div class="space-y-1">
								<div class="flex items-center gap-2">
									<p class="text-sm font-medium text-zinc-900 dark:text-zinc-100">{item.name}</p>
									<span class={`rounded-full px-2 py-0.5 text-[11px] font-medium ${
										item.isEnabled
											? 'bg-riptide-100 text-riptide-800 dark:bg-riptide-950/40 dark:text-riptide-200'
											: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300'
									}`}>
										{item.isEnabled ? m.user_image_beds_enabled() : m.user_image_beds_disabled()}
									</span>
								</div>
								<p class="text-xs text-zinc-500 dark:text-zinc-400">
									{item.providerType === 'lsky'
										? m.user_image_beds_provider_lsky()
										: m.user_image_beds_provider_see()}
								</p>
								{#if item.baseUrl}
									<p class="text-xs text-zinc-500 dark:text-zinc-400">{item.baseUrl}</p>
								{/if}
								{#if item.strategyId}
									<p class="text-xs text-zinc-500 dark:text-zinc-400">
										{m.user_image_beds_strategy_label()}: {item.strategyId}
									</p>
								{/if}
								{#if item.storageId}
									<p class="text-xs text-zinc-500 dark:text-zinc-400">
										{m.user_image_beds_storage_id_label()}: {item.storageId}
									</p>
								{/if}
							</div>
							<div class="flex gap-2">
								<button
									type="button"
									class="rounded-lg border border-zinc-200 px-3 py-1.5 text-sm text-zinc-700 transition hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-900"
									onclick={() => startEdit(item)}
								>
									{m.common_edit()}
								</button>
								<button
									type="button"
									class="rounded-lg border border-red-200 px-3 py-1.5 text-sm text-red-700 transition hover:bg-red-50 disabled:opacity-60 dark:border-red-900/60 dark:text-red-300 dark:hover:bg-red-950/30"
									onclick={() => handleDelete(item.id)}
									disabled={deletingId === item.id}
								>
									{deletingId === item.id ? m.common_deleting() : m.common_delete()}
								</button>
							</div>
						</div>
					</div>
				{/each}
			{/if}
		</div>
	</div>

	<div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950">
		<div class="space-y-1">
			<h2 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
				{editingId ? m.user_image_beds_edit_title() : m.user_image_beds_create_title()}
			</h2>
			<p class="text-sm text-zinc-500 dark:text-zinc-400">
				{m.user_image_beds_form_description()}
			</p>
		</div>

		<form class="mt-5 space-y-4 sm:max-w-2xl" onsubmit={handleSubmit}>
			<div class="grid gap-4 sm:grid-cols-2">
				<div class="space-y-1">
					<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for="image-bed-name">
						{m.user_image_beds_name_label()}
					</label>
					<input
						id="image-bed-name"
						bind:value={form.name}
						type="text"
						class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
						placeholder={m.user_image_beds_name_placeholder()}
					/>
				</div>

				<div class="space-y-1">
					<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for="image-bed-provider">
						{m.user_image_beds_provider_label()}
					</label>
					<select
						id="image-bed-provider"
						bind:value={form.providerType}
						class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
					>
						<option value="see">{m.user_image_beds_provider_see()}</option>
						<option value="lsky">{m.user_image_beds_provider_lsky()}</option>
					</select>
				</div>
			</div>

			{#if form.providerType === 'lsky'}
				<div class="space-y-1">
					<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for="image-bed-base-url">
						{m.user_image_beds_base_url_label()}
					</label>
					<input
						id="image-bed-base-url"
						bind:value={form.baseUrl}
						type="url"
						class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
						placeholder={m.user_image_beds_base_url_placeholder()}
					/>
				</div>
			{/if}

			<div class="space-y-1">
				<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for="image-bed-api-token">
					{m.user_image_beds_api_token_label()}
				</label>
				<input
					id="image-bed-api-token"
					bind:value={form.apiToken}
					type="password"
					class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
					placeholder={m.user_image_beds_api_token_placeholder()}
				/>
			</div>

			{#if form.providerType === 'lsky'}
				<div class="grid gap-4 sm:grid-cols-2">
					<div class="space-y-1">
						<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for="image-bed-storage-id">
							{m.user_image_beds_storage_id_label()}
						</label>
						<input
							id="image-bed-storage-id"
							bind:value={form.storageId}
							type="number"
							min="1"
							step="1"
							class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
							placeholder={m.user_image_beds_storage_id_placeholder()}
						/>
					</div>
				</div>

				<div class="space-y-1">
					<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for="image-bed-strategy-id">
						{m.user_image_beds_strategy_label()}
					</label>
					<input
						id="image-bed-strategy-id"
						bind:value={form.strategyId}
						type="text"
						class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
						placeholder={m.user_image_beds_strategy_placeholder()}
					/>
				</div>
			{/if}

			<label class="flex items-center gap-3 rounded-xl border border-zinc-200 px-4 py-3 text-sm text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">
				<input bind:checked={form.isEnabled} type="checkbox" class="h-4 w-4 rounded border-zinc-300 text-riptide-500 focus:ring-riptide-400" />
				<span>{m.user_image_beds_enabled_toggle()}</span>
			</label>

			<div class="flex flex-wrap justify-end gap-3">
				<button
					type="button"
					class="rounded-lg border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-900"
					onclick={startCreate}
				>
					{m.common_reset()}
				</button>
				<button
					type="submit"
					class="rounded-lg bg-zinc-900 px-5 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
					disabled={saving}
				>
					{saving ? m.common_saving() : m.common_save()}
				</button>
			</div>
		</form>
	</div>
</div>
