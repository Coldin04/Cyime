<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
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
	let editingId = $state<string | null>(null);
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
								<p class="text-xs text-zinc-500 dark:text-zinc-400">{getProviderDisplayName(item.providerType)}</p>
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
						class="w-full rounded-lg border border-zinc-200 bg-transparent px-4 py-2 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-800 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
						placeholder={getFieldPlaceholder(field)}
					/>
					{#if getFieldHelpText(field)}
						<p class="text-xs text-zinc-500 dark:text-zinc-400">{getFieldHelpText(field)}</p>
					{/if}
				</div>
			{/each}

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
