<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import { portal } from '$lib/actions/portal';
	import ConfirmDialog from '$lib/components/common/ConfirmDialog.svelte';
	import {
		createImageBedConfig,
		deleteImageBedConfig,
		getImageBedConfigs,
		getImageBedProviders,
		updateImageBedConfig,
		type ImageBedConfig,
		type ImageBedProvider,
		type ImageBedProviderField,
		type UpsertImageBedConfigRequest
	} from '$lib/api/user';
	import X from '~icons/ph/x';
	import PencilSimple from '~icons/ph/pencil-simple';
	import Trash from '~icons/ph/trash';

	type ProviderType = string;

	type FormState = {
		name: string;
		providerType: ProviderType;
		isEnabled: boolean;
		fieldValues: Record<string, string>;
	};

	const emptyForm = (): FormState => ({
		name: '',
		providerType: 'see',
		isEnabled: true,
		fieldValues: {}
	});

	let loading = $state(false);
	let saving = $state(false);
	let deletingId = $state<string | null>(null);
	let deleteCandidateId = $state<string | null>(null);
	let togglingId = $state<string | null>(null);
	let editingId = $state<string | null>(null);
	let formDialogOpen = $state(false);
	let items = $state<ImageBedConfig[]>([]);
	let providers = $state<ImageBedProvider[]>([]);
	let form = $state<FormState>(emptyForm());

	onMount(() => {
		void loadInitialData();
	});

	async function loadInitialData() {
		loading = true;
		try {
			const [loadedProviders, loadedConfigs] = await Promise.all([
				getImageBedProviders(),
				getImageBedConfigs()
			]);
			providers = loadedProviders;
			items = loadedConfigs;
			if (providers.length > 0 && !providers.some((provider) => provider.providerType === form.providerType)) {
				form.providerType = providers[0].providerType;
			}
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_image_beds_load_failed());
		} finally {
			loading = false;
		}
	}

	function startCreate() {
		editingId = null;
		form = emptyForm();
		if (providers.length > 0) {
			form.providerType = providers[0].providerType;
		}
	}

	function openCreateDialog() {
		startCreate();
		formDialogOpen = true;
	}

	function openEditDialog(item: ImageBedConfig) {
		startEdit(item);
		formDialogOpen = true;
	}

	function closeFormDialog() {
		if (saving) return;
		formDialogOpen = false;
	}

	function startEdit(item: ImageBedConfig) {
		editingId = item.id;
		const fieldValues: Record<string, string> = {
			...(item.fieldValues ?? {})
		};
		if (!fieldValues.baseUrl && item.baseUrl) {
			fieldValues.baseUrl = item.baseUrl;
		}
		// Token is intentionally not returned from server; user can re-enter to rotate it.
		if (!fieldValues.storageId && item.storageId) {
			fieldValues.storageId = String(item.storageId);
		}
		if (!fieldValues.strategyId && item.strategyId) {
			fieldValues.strategyId = item.strategyId;
		}

		form = {
			name: item.name,
			providerType: item.providerType,
			isEnabled: item.isEnabled,
			fieldValues
		};
	}

	const currentProvider = $derived(
		providers.find((provider) => provider.providerType === form.providerType) ?? null
	);

	const currentProviderFields = $derived(currentProvider?.fields ?? []);

	function toRequestBody(input: FormState): UpsertImageBedConfigRequest {
		const fieldValues = { ...input.fieldValues };
		const storageID = Number.parseInt(fieldValues.storageId ?? '', 10);
		return {
			name: input.name,
			providerType: input.providerType,
			baseUrl: fieldValues.baseUrl ?? '',
			apiToken: fieldValues.apiToken ?? '',
			isEnabled: input.isEnabled,
			storageId: Number.isNaN(storageID) ? 0 : storageID,
			strategyId: fieldValues.strategyId ?? '',
			fieldValues
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
			formDialogOpen = false;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_image_beds_update_failed());
		} finally {
			saving = false;
		}
	}

	async function handleDelete(id: string) {
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

	function openDeleteConfirm(id: string) {
		deleteCandidateId = id;
	}

	function closeDeleteConfirm() {
		if (deletingId) return;
		deleteCandidateId = null;
	}

	async function confirmDelete() {
		if (!deleteCandidateId) return;
		const targetId = deleteCandidateId;
		deleteCandidateId = null;
		await handleDelete(targetId);
	}

	function toUpdateRequestFromItem(item: ImageBedConfig, isEnabled: boolean): UpsertImageBedConfigRequest {
		const fieldValues: Record<string, string> = {
			...(item.fieldValues ?? {})
		};
		if (!fieldValues.baseUrl && item.baseUrl) {
			fieldValues.baseUrl = item.baseUrl;
		}
		if (!fieldValues.storageId && item.storageId) {
			fieldValues.storageId = String(item.storageId);
		}
		if (!fieldValues.strategyId && item.strategyId) {
			fieldValues.strategyId = item.strategyId;
		}

		return {
			name: item.name,
			providerType: item.providerType,
			baseUrl: fieldValues.baseUrl ?? item.baseUrl ?? '',
			apiToken: fieldValues.apiToken ?? '',
			isEnabled,
			storageId: item.storageId ?? 0,
			strategyId: fieldValues.strategyId ?? item.strategyId ?? '',
			fieldValues
		};
	}

	async function handleToggleEnabled(item: ImageBedConfig) {
		togglingId = item.id;
		try {
			const updated = await updateImageBedConfig(item.id, toUpdateRequestFromItem(item, !item.isEnabled));
			items = items.map((row) => (row.id === updated.id ? updated : row));
			toast.success(updated.isEnabled ? m.user_image_beds_enabled_now() : m.user_image_beds_disabled_now());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_image_beds_update_failed());
		} finally {
			togglingId = null;
		}
	}

	function getProviderDisplayName(providerType: string): string {
		return providers.find((provider) => provider.providerType === providerType)?.displayName ?? providerType;
	}

	function getFieldValue(fieldKey: string): string {
		return form.fieldValues[fieldKey] ?? '';
	}

	function setFieldValue(fieldKey: string, value: string) {
		const trimmed = value.trim();
		if (trimmed === '') {
			delete form.fieldValues[fieldKey];
			form.fieldValues = { ...form.fieldValues };
			return;
		}
		form.fieldValues[fieldKey] = value;
		form.fieldValues = { ...form.fieldValues };
	}

	type MessageFn = (args?: Record<string, unknown>, options?: Record<string, unknown>) => string;
	const messageMap = m as unknown as Record<string, MessageFn>;

	function resolveI18nKey(key: string | undefined, fallback: string): string {
		if (!key) {
			return fallback;
		}
		const fn = messageMap[key];
		if (!fn) {
			return fallback;
		}
		try {
			return fn();
		} catch {
			return fallback;
		}
	}

	function getFieldLabel(field: ImageBedProviderField): string {
		return resolveI18nKey(field.labelKey, field.label);
	}

	function getFieldPlaceholder(field: ImageBedProviderField): string {
		return resolveI18nKey(field.placeholderKey, field.placeholder ?? '');
	}

	function getFieldHelpText(field: ImageBedProviderField): string {
		return resolveI18nKey(field.helpTextKey, field.helpText ?? '');
	}

	type SupportedInputMode =
		| 'none'
		| 'text'
		| 'tel'
		| 'url'
		| 'email'
		| 'numeric'
		| 'decimal'
		| 'search'
		| undefined;

	function normalizeInputMode(value: string | undefined): SupportedInputMode {
		switch (value) {
			case 'none':
			case 'text':
			case 'tel':
			case 'url':
			case 'email':
			case 'numeric':
			case 'decimal':
			case 'search':
				return value;
			default:
				return undefined;
		}
	}
</script>

<div class="space-y-4">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
		<div class="space-y-1">
			<h1 class="text-2xl font-bold text-zinc-900 dark:text-zinc-100">
				{m.user_image_beds_title()}
			</h1>
			<p class="text-sm text-zinc-500 dark:text-zinc-400">
				{m.user_image_beds_description()}
			</p>
		</div>
		<button
			type="button"
			class="shrink-0 rounded-md border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-900"
			onclick={openCreateDialog}
		>
			{m.user_image_beds_add()}
		</button>
	</div>

	{#if loading}
		<p class="text-sm text-zinc-500 dark:text-zinc-400">{m.workspace_loading()}</p>
	{:else if items.length === 0}
		<div class="rounded-xl border border-dashed border-zinc-300 px-4 py-6 text-sm text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
			{m.user_image_beds_empty()}
		</div>
	{:else}
			<div class="divide-y divide-zinc-200 border-y border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
				{#each items as item (item.id)}
					<div class="py-4">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
							<div class="space-y-1">
								<div class="flex items-center gap-2">
									<p class="text-base font-medium text-zinc-900 dark:text-zinc-100">{item.name}</p>
								<span class={`rounded-full px-2 py-0.5 text-[11px] font-medium ${
									item.isEnabled
										? 'bg-riptide-100 text-riptide-800 dark:bg-riptide-950/40 dark:text-riptide-200'
										: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300'
								}`}>
									{item.isEnabled ? m.user_image_beds_enabled() : m.user_image_beds_disabled()}
								</span>
							</div>
							<p class="text-sm text-zinc-500 dark:text-zinc-400">{getProviderDisplayName(item.providerType)}</p>
							{#if item.baseUrl}
								<p class="text-sm text-zinc-500 dark:text-zinc-400">{item.baseUrl}</p>
							{/if}
							{#if item.strategyId}
								<p class="text-sm text-zinc-500 dark:text-zinc-400">
									{m.user_image_beds_strategy_label()}: {item.strategyId}
								</p>
							{/if}
							{#if item.storageId}
								<p class="text-sm text-zinc-500 dark:text-zinc-400">
									{m.user_image_beds_storage_id_label()}: {item.storageId}
								</p>
							{/if}
						</div>
							<div class="flex items-center">
								<div class="flex items-center gap-0.5">
									<button
										type="button"
										class="grid h-7 w-7 place-content-center rounded text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 disabled:opacity-60 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
										onclick={() => openEditDialog(item)}
										aria-label={m.common_edit()}
										title={m.common_edit()}
									>
										<PencilSimple class="h-4 w-4" />
									</button>
									<button
										type="button"
										class="grid h-7 w-7 place-content-center rounded text-red-500 transition hover:bg-red-50 hover:text-red-700 disabled:opacity-60 dark:text-red-300 dark:hover:bg-red-950/30 dark:hover:text-red-200"
										onclick={() => openDeleteConfirm(item.id)}
										disabled={deletingId === item.id}
										aria-label={m.common_delete()}
										title={m.common_delete()}
									>
										{#if deletingId === item.id}
											<span class="h-4 w-4 text-[10px] leading-4">...</span>
										{:else}
											<Trash class="h-4 w-4" />
										{/if}
									</button>
								</div>
								<button
									type="button"
									class={`relative ml-2 inline-flex h-6 w-10 shrink-0 items-center rounded-full transition-colors disabled:opacity-60 ${
										item.isEnabled
											? 'bg-riptide-500 hover:bg-riptide-600'
											: 'bg-zinc-300 hover:bg-zinc-400 dark:bg-zinc-700 dark:hover:bg-zinc-600'
								}`}
								onclick={() => handleToggleEnabled(item)}
								disabled={togglingId === item.id}
								aria-label={item.isEnabled ? m.user_image_beds_disable_action() : m.user_image_beds_enable_action()}
								title={item.isEnabled ? m.user_image_beds_disable_action() : m.user_image_beds_enable_action()}
							>
								<span
									class={`inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
										item.isEnabled ? 'translate-x-5' : 'translate-x-1'
									}`}
								></span>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	{#if formDialogOpen}
		<div
			use:portal
			class="fixed inset-0 z-[130] bg-black/45 p-4"
			role="presentation"
			onclick={closeFormDialog}
		>
			<div class="flex min-h-full items-center justify-center">
				<div
					class="w-full max-w-2xl overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-950"
					role="dialog"
					aria-modal="true"
					aria-label={editingId ? m.user_image_beds_edit_title() : m.user_image_beds_create_title()}
					tabindex="-1"
					onclick={(event) => event.stopPropagation()}
					onkeydown={(event) => {
						if (event.key === 'Escape') closeFormDialog();
					}}
				>
					<header class="flex h-14 items-center justify-between border-b border-zinc-200 px-5 dark:border-zinc-800">
						<h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
							{editingId ? m.user_image_beds_edit_title() : m.user_image_beds_create_title()}
						</h3>
						<button
							type="button"
							class="rounded-full p-2 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
							onclick={closeFormDialog}
						>
							<X class="h-4 w-4" />
						</button>
					</header>

					<form class="space-y-4 p-5 sm:p-6" onsubmit={handleSubmit}>
						<p class="text-sm text-zinc-500 dark:text-zinc-400">
							{m.user_image_beds_form_description()}
						</p>

						<div class="grid gap-4 sm:grid-cols-2">
							<div class="space-y-1">
								<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for="image-bed-name">
									{m.user_image_beds_name_label()}
								</label>
								<input
									id="image-bed-name"
									bind:value={form.name}
									type="text"
									class="w-full rounded-md border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
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
									class="w-full rounded-md border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
								>
									{#each providers as provider (provider.providerType)}
										<option value={provider.providerType}>{provider.displayName}</option>
									{/each}
								</select>
							</div>
						</div>

						{#each currentProviderFields as field (field.key)}
							<div class="space-y-1">
								<label class="text-sm font-medium text-zinc-900 dark:text-zinc-100" for={`image-bed-field-${field.key}`}>
									{getFieldLabel(field)}
								</label>
								<input
									id={`image-bed-field-${field.key}`}
									value={getFieldValue(field.key) ?? ''}
									oninput={(event) => setFieldValue(field.key, event.currentTarget.value)}
									type={field.type === 'number' ? 'number' : field.type}
									inputmode={normalizeInputMode(field.inputMode)}
									min={field.type === 'number' ? '1' : undefined}
									step={field.type === 'number' ? '1' : undefined}
									required={field.required}
									class="w-full rounded-md border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
									placeholder={getFieldPlaceholder(field)}
								/>
								{#if getFieldHelpText(field)}
									<p class="text-xs text-zinc-500 dark:text-zinc-400">{getFieldHelpText(field)}</p>
								{/if}
							</div>
						{/each}

						<label class="flex items-center gap-3 rounded-md border border-zinc-200 px-4 py-3 text-sm text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">
							<input
								bind:checked={form.isEnabled}
								type="checkbox"
								class="h-4 w-4 rounded border-zinc-300 text-riptide-500 focus:ring-riptide-400"
							/>
							<span>{m.user_image_beds_enabled_toggle()}</span>
						</label>

						<div class="flex flex-wrap justify-end gap-3">
							<button
								type="button"
								class="rounded-md border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-900"
								onclick={closeFormDialog}
							>
								{m.common_cancel()}
							</button>
							<button
								type="submit"
								class="rounded-md bg-zinc-900 px-5 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
								disabled={saving}
							>
								{saving ? m.common_saving() : m.common_save()}
							</button>
						</div>
					</form>
				</div>
			</div>
		</div>
	{/if}
</div>

<ConfirmDialog
	open={deleteCandidateId !== null}
	title={m.common_delete()}
	message={m.user_image_beds_confirm_delete()}
	confirmText={m.common_delete()}
	onCancel={closeDeleteConfirm}
	onConfirm={() => void confirmDelete()}
/>
