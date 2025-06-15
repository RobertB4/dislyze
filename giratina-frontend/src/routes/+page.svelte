<script lang="ts">
	import Layout from "$components/Layout.svelte";
	import {
		Badge,
		Tooltip,
		Slideover,
		Input,
		InteractivePill,
		Alert,
		Button,
		toast
	} from "@dislyze/zoroark";
	import type { PageData } from "./$types";
	import { createForm } from "felte";
	import { invalidate } from "$app/navigation";
	import { mutationFetch } from "$lib/fetch";
	import type { Tenant } from "./+page";
	import type { EnterpriseFeatures } from "@dislyze/zoroark";

	let { data: pageData }: { data: PageData } = $props();

	const featureKeyToLabelMap: Record<string, string> = {
		rbac: "権限設定"
	};

	interface UpdateTenantRequestBody {
		name: string;
		enterprise_features: EnterpriseFeatures;
	}

	let editingTenant = $state<Tenant | null>(null);

	let isInviteSlideoverOpen = $state(false);
	let generatedInviteUrl = $state<string | null>(null);

	const {
		form: editForm,
		data: editData,
		errors: editErrors,
		isSubmitting: editIsSubmitting,
		reset: editReset,
		setInitialValues: setEditFormInitialValues
	} = createForm({
		initialValues: {
			name: "",
			rbac_enabled: false
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.name = values.name.trim();

			if (!values.name) {
				errs.name = "テナント名は必須です";
			}

			return errs;
		},
		onSubmit: async (values) => {
			if (!editingTenant) return;

			const requestBody: UpdateTenantRequestBody = {
				name: values.name,
				enterprise_features: {
					rbac: { enabled: values.rbac_enabled }
				}
			};

			const { success } = await mutationFetch(`/api/tenants/${editingTenant.id}/update`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify(requestBody)
			});

			if (success) {
				await invalidate((u) => u.pathname === "/api/tenants");
				editReset();
				toast.show("テナントを更新しました。", "success");
				editingTenant = null;
			}
		}
	});

	const handleEditTenant = (tenant: Tenant) => {
		setEditFormInitialValues({
			name: tenant.name,
			rbac_enabled: tenant.enterprise_features.rbac.enabled
		});
		editingTenant = tenant;
	};

	const handleEditClose = () => {
		editingTenant = null;
		editReset();
	};

	const {
		form: inviteForm,
		data: inviteData,
		errors: inviteErrors,
		isSubmitting: inviteIsSubmitting,
		reset: inviteReset
	} = createForm({
		initialValues: {
			email: "",
			company_name: "",
			user_name: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.email = values.email.trim();
			values.company_name = values.company_name.trim();
			values.user_name = values.user_name.trim();

			if (!values.email) {
				errs.email = "メールアドレスは必須です";
			} else if (!values.email.includes("@")) {
				errs.email = "有効なメールアドレスを入力してください";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const { response, success } = await mutationFetch("/api/tenants/generate-token", {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					email: values.email,
					company_name: values.company_name,
					user_name: values.user_name
				})
			});

			if (success && response) {
				const data = await response.json();
				generatedInviteUrl = data.url;
				toast.show("招待URLを生成しました。", "success");
			}
		}
	});

	const handleInviteOpen = () => {
		isInviteSlideoverOpen = true;
		generatedInviteUrl = null;
		inviteReset();
	};

	const handleInviteClose = () => {
		isInviteSlideoverOpen = false;
		generatedInviteUrl = null;
		inviteReset();
	};

	const copyToClipboard = async () => {
		if (generatedInviteUrl) {
			try {
				await navigator.clipboard.writeText(generatedInviteUrl);
				toast.show("URLをクリップボードにコピーしました。", "success");
			} catch (err) {
				console.error("Failed to copy URL to clipboard:", err);
				toast.show("コピーに失敗しました。", "error");
			}
		}
	};
</script>

<Layout
	me={pageData.me}
	pageTitle="テナント一覧"
	promises={{
		tenantsResponse: pageData.tenantsPromise
	}}
>
	{#snippet buttons()}
		<Button
			type="button"
			variant="primary"
			onclick={handleInviteOpen}
			data-testid="generate-tenant-invitation-token-button"
		>
			テナントを招待
		</Button>
	{/snippet}

	{#snippet skeleton()}
		<div class="animate-pulse">
			<div class="mt-8 flow-root">
				<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
					<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
						<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
							<div class="min-w-full divide-y divide-gray-300">
								<div class="bg-gray-50 px-6 py-3">
									<div class="h-4 bg-gray-300 rounded w-1/4"></div>
								</div>
								{#each Array(5)}
									<div class="bg-white px-6 py-4">
										<div class="space-y-2">
											<div class="h-4 bg-gray-200 rounded w-1/3"></div>
											<div class="h-3 bg-gray-200 rounded w-1/2"></div>
										</div>
									</div>
								{/each}
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/snippet}

	{#snippet children({ tenantsResponse })}
		{@const { tenants } = tenantsResponse}

		{#if isInviteSlideoverOpen}
			<form
				use:inviteForm
				class="space-y-6 p-1 flex flex-col h-full"
				data-testid="invite-tenant-form"
			>
				<Slideover
					title="テナントを招待"
					onClose={handleInviteClose}
					data-testid="invite-tenant-slideover"
				>
					<div class="flex-grow space-y-6">
						<!-- 48-hour validity alert at the top -->
						<Alert type="info" title="注意" data-testid="invite-validity-alert">
							<p>招待リンクの有効期限は48時間です。</p>
						</Alert>

						<!-- Company name input field (optional) -->
						<Input
							id="invite-company-name"
							name="company_name"
							type="text"
							label="会社名（任意）"
							bind:value={$inviteData.company_name}
							error={$inviteErrors.company_name?.[0]}
							placeholder="株式会社Example"
							variant="underlined"
						/>

						<!-- User name input field (optional) -->
						<Input
							id="invite-user-name"
							name="user_name"
							type="text"
							label="氏名（任意）"
							bind:value={$inviteData.user_name}
							error={$inviteErrors.user_name?.[0]}
							placeholder="田中太郎"
							variant="underlined"
						/>

						<!-- Email input field -->
						<Input
							id="invite-email"
							name="email"
							type="email"
							label="招待先メールアドレス"
							bind:value={$inviteData.email}
							error={$inviteErrors.email?.[0]}
							required
							placeholder="user@example.com"
							variant="underlined"
						/>

						<!-- Generate token button (not primary button of slideover) -->
						<Button
							type="submit"
							variant="primary"
							loading={$inviteIsSubmitting}
							disabled={$inviteIsSubmitting || !$inviteData.email.trim()}
							data-testid="generate-token-button"
						>
							招待URLを生成
						</Button>

						<!-- Generated URL display area (only shown when URL exists) -->
						{#if generatedInviteUrl}
							<div class="space-y-3">
								<h3 class="text-sm font-medium text-gray-700">招待URL</h3>
								<div class="flex items-center gap-3 p-3 bg-gray-50 rounded-md shadow-md">
									<div class="flex-1 font-mono text-sm text-gray-800 break-all">
										{generatedInviteUrl}
									</div>
									<Button
										type="button"
										variant="secondary"
										onclick={copyToClipboard}
										data-testid="copy-url-button"
									>
										コピー
									</Button>
								</div>
							</div>
						{/if}
					</div>
				</Slideover>
			</form>
		{/if}

		{#if editingTenant}
			<form use:editForm class="space-y-6 p-1 flex flex-col h-full" data-testid="edit-tenant-form">
				<Slideover
					title="テナントを編集"
					primaryButtonText="更新"
					primaryButtonTypeSubmit={true}
					onClose={handleEditClose}
					loading={$editIsSubmitting}
					data-testid="edit-tenant-slideover"
				>
					<div class="flex-grow space-y-6">
						<Input
							id="edit-tenant-name"
							name="name"
							type="text"
							label="テナント名"
							bind:value={$editData.name}
							error={$editErrors.name?.[0]}
							required
							placeholder="テナント名を入力"
							variant="underlined"
						/>

						<div class="space-y-4">
							<h3 class="text-sm font-medium text-gray-700">エンタープライズ機能</h3>

							<div class="border border-gray-200 rounded-lg p-4">
								<div class="flex items-center justify-between">
									<h4 class="text-sm font-medium text-gray-900">権限設定 (RBAC)</h4>
									<div class="flex gap-2">
										<InteractivePill
											selected={!$editData.rbac_enabled}
											onclick={() => ($editData.rbac_enabled = false)}
											variant="orange"
											data-testid="rbac-disabled"
										>
											無効
										</InteractivePill>
										<InteractivePill
											selected={$editData.rbac_enabled}
											onclick={() => ($editData.rbac_enabled = true)}
											variant="orange"
											data-testid="rbac-enabled"
										>
											有効
										</InteractivePill>
									</div>
								</div>
							</div>
						</div>
					</div>
				</Slideover>
			</form>
		{/if}

		<div class="mt-8 flow-root">
			{#if tenants.length === 0}
				<div class="text-center py-12" data-testid="no-tenants-message">
					<div class="text-gray-500 text-lg">テナントが見つかりませんでした</div>
				</div>
			{:else}
				<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
					<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
						<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
							<table class="min-w-full divide-y divide-gray-300" data-testid="tenants-table">
								<thead class="bg-gray-50" data-testid="tenants-table-header">
									<tr>
										<th
											scope="col"
											class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6"
											data-testid="tenant-table-header-name">テナント名</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="tenant-table-header-id">ID</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="tenant-table-header-stripe">Stripe ID</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="tenant-table-header-features">エンタープライズ機能</th
										>
										<th
											scope="col"
											class="relative py-3.5 pl-3 pr-4 sm:pr-6"
											data-testid="tenant-table-header-actions"
										>
											<span class="sr-only">操作</span>
										</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-200 bg-white" data-testid="tenants-table-body">
									{#each tenants as tenant (tenant.id)}
										{@const enabledFeatures = Object.entries(tenant.enterprise_features)
											.filter(([, feature]) => feature.enabled)
											.map(([feature]) => feature)}
										<tr data-testid={`tenant-row-${tenant.id}`}>
											<td
												class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
												data-testid={`tenant-name-${tenant.id}`}>{tenant.name}</td
											>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 font-mono"
												data-testid={`tenant-id-${tenant.id}`}
											>
												{tenant.id}
											</td>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 font-mono"
												data-testid={`tenant-stripe-${tenant.id}`}
											>
												{#if tenant.stripe_customer_id}
													{tenant.stripe_customer_id}
												{:else}
													<span class="text-gray-400">未設定</span>
												{/if}
											</td>
											<td
												class="px-3 py-4 text-sm text-gray-500"
												data-testid={`tenant-features-${tenant.id}`}
											>
												{#if enabledFeatures.length === 0}
													<span class="text-gray-400">なし</span>
												{:else}
													<div class="flex flex-wrap gap-1 items-center">
														{#each enabledFeatures.slice(0, 3) as feature (feature)}
															<Badge color="blue" size="sm" rounded="md">
																{featureKeyToLabelMap[feature]}
															</Badge>
														{/each}
														{#if enabledFeatures.length > 3}
															<Tooltip class="ml-2">
																{#snippet content()}
																	<div class="space-y-1">
																		{#each enabledFeatures.slice(3) as feature (feature)}
																			<div class="text-xs">{featureKeyToLabelMap[feature]}</div>
																		{/each}
																	</div>
																{/snippet}

																<span
																	class="text-gray-400 cursor-help border-b border-dotted border-gray-300 flex items-center"
																	data-testid={`tenant-features-overflow-${tenant.id}`}
																>
																	他{enabledFeatures.length - 3}件
																	<svg
																		class="h-5 w-5 text-gray-800 ml-1"
																		fill="currentColor"
																		viewBox="0 0 20 20"
																	>
																		<path
																			fill-rule="evenodd"
																			d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 00-.867.5 1 1 0 11-1.731-1A3 3 0 0113 8a3.001 3.001 0 01-2 2.83V11a1 1 0 11-2 0v-1a1 1 0 011-1 1 1 0 100-2zm0 8a1 1 0 100-2 1 1 0 000 2z"
																			clip-rule="evenodd"
																		></path>
																	</svg>
																</span>
															</Tooltip>
														{/if}
													</div>
												{/if}
											</td>
											<td
												class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
												data-testid={`tenant-actions-${tenant.id}`}
											>
												<Button
													type="button"
													variant="link"
													onclick={() => handleEditTenant(tenant)}
													data-testid={`edit-tenant-button-${tenant.id}`}
												>
													編集
												</Button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				</div>
			{/if}
		</div>
	{/snippet}
</Layout>
