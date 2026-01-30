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
	import type { Tenant, EnterpriseFeatures } from "./+page";
	import { resolve } from "$app/paths";

	let { data: pageData }: { data: PageData } = $props();

	const featureKeyToLabelMap: Record<string, string> = {
		rbac: "権限設定",
		ip_whitelist: "IPアドレス制限",
		sso: "SSO認証"
	};

	const isFeatureEditable = (featureKey: string): boolean => {
		return featureKey !== "sso";
	};

	interface UpdateTenantRequestBody {
		name: string;
		enterprise_features: EnterpriseFeatures;
	}

	let editingTenant = $state<Tenant | null>(null);

	let isInviteSlideoverOpen = $state(false);
	let generatedInviteUrl = $state<string | null>(null);
	let loginTenant = $state<Tenant | null>(null);

	const {
		form: editForm,
		data: editData,
		errors: editErrors,
		isSubmitting: editIsSubmitting,
		reset: editReset,
		setInitialValues: setEditFormInitialValues
	} = createForm<{ name: string; enterprise_features: EnterpriseFeatures }>({
		initialValues: {
			name: "",
			enterprise_features: {} as EnterpriseFeatures
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
				enterprise_features: values.enterprise_features
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
			enterprise_features: tenant.enterprise_features
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
		reset: inviteReset,
		addField: addInviteField,
		unsetField: unsetInviteField
	} = createForm({
		initialValues: {
			email: "",
			company_name: "",
			user_name: "",
			sso_enabled: false,
			idp_metadata_url: "",
			allowed_domains: [{ value: "" }]
		},
		validate: (values) => {
			const errs: Record<string, string | { value: string }[]> = {};
			values.email = values.email?.trim() || "";
			values.company_name = values.company_name?.trim() || "";
			if (values.user_name !== undefined) {
				values.user_name = values.user_name.trim();
			}

			if (!values.email) {
				errs.email = "メールアドレスは必須です";
			} else if (!values.email.includes("@")) {
				errs.email = "有効なメールアドレスを入力してください";
			}

			if (values.sso_enabled) {
				if (values.idp_metadata_url !== undefined) {
					values.idp_metadata_url = values.idp_metadata_url.trim();
				}

				if (!values.idp_metadata_url) {
					errs.idp_metadata_url = "IdPメタデータURLは必須です";
				} else {
					try {
						new URL(values.idp_metadata_url);
					} catch {
						errs.idp_metadata_url = "有効なURLを入力してください";
					}
				}

				if (values.allowed_domains && Array.isArray(values.allowed_domains)) {
					const hasAtLeastOneDomain = values.allowed_domains.some((d) => d?.value?.trim() !== "");
					if (!hasAtLeastOneDomain && values.allowed_domains.length > 0) {
						errs.allowed_domains = [{ value: "少なくとも1つの許可ドメインが必要です" }];
					}

					const domainErrors: { value?: string }[] = [];
					values.allowed_domains.forEach((domain) => {
						if (domain?.value?.trim()) {
							const trimmedDomain = domain.value.trim();
							if (trimmedDomain.startsWith("http://") || trimmedDomain.startsWith("https://")) {
								domainErrors.push({ value: "プロトコル（http://、https://）を含めないでください" });
							} else {
								domainErrors.push({});
							}
						} else {
							domainErrors.push({});
						}
					});

					if (domainErrors.some((err) => err.value)) {
						errs.allowed_domains = domainErrors;
					}
				}
			}

			return errs;
		},
		onSubmit: async (values) => {
			const body: any = {
				email: values.email,
				company_name: values.company_name,
				user_name: values.user_name
			};

			if (values.sso_enabled) {
				body.sso = {
					enabled: true,
					idp_metadata_url: values.idp_metadata_url,
					allowed_domains: values.allowed_domains
						.filter((d) => d?.value?.trim() !== "")
						.map((d) => d.value.trim())
				};
			}

			const { response, success } = await mutationFetch("/api/tenants/generate-token", {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify(body)
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

	const addAllowedDomain = () => {
		addInviteField("allowed_domains", { value: "" });
	};

	const removeAllowedDomain = (index: number) => {
		if ($inviteData.allowed_domains.length > 1) {
			unsetInviteField(`allowed_domains.${index}`);
		}
	};

	const isInviteFormValid = $derived(() => {
		const emailValid = $inviteData.email?.trim() !== "";

		if (!$inviteData.sso_enabled) {
			return emailValid;
		}

		const metadataUrlValid = $inviteData.idp_metadata_url?.trim() !== "";
		const hasAtLeastOneDomain = $inviteData.allowed_domains.some((d) => d?.value?.trim() !== "");

		return emailValid && metadataUrlValid && hasAtLeastOneDomain;
	});

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

	const {
		form: loginForm,
		data: loginData,
		errors: loginErrors,
		isSubmitting: loginIsSubmitting,
		reset: loginReset
	} = createForm({
		initialValues: {
			tenant_name: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.tenant_name = values.tenant_name.trim();

			if (values.tenant_name !== loginTenant?.name) {
				errs.tenant_name = "正しいテナント名を入力してください";
			}

			return errs;
		},
		onSubmit: () => {
			if (!loginTenant) return;

			// Navigate directly to the endpoint - server will set cookies and redirect
			window.location.href = `/api/tenants/${loginTenant.id}/login`;
		}
	});

	const handleLoginOpen = (tenant: Tenant) => {
		loginTenant = tenant;
		loginReset();
	};

	const handleLoginClose = () => {
		loginTenant = null;
		loginReset();
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

						<!-- User name input field -->
						<Input
							id="invite-user-name"
							name="user_name"
							type="text"
							label="氏名（任意）"
							value={$inviteData.user_name ?? ""}
							oninput={(e) => {
								const target = e.currentTarget as HTMLInputElement;
								$inviteData.user_name = target.value;
							}}
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

						<!-- SSO Toggle -->
						<div class="space-y-2">
							<label class="flex items-center cursor-pointer">
								<input
									type="checkbox"
									bind:checked={$inviteData.sso_enabled}
									onchange={() => {
										if ($inviteData.sso_enabled && $inviteData.allowed_domains.length === 0) {
											addInviteField("allowed_domains", { value: "" });
										}
									}}
									class="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
									data-testid="sso-enabled-toggle"
								/>
								<span class="ml-2 text-sm font-medium text-gray-700">SSO認証を有効にする</span>
							</label>
						</div>

						{#if $inviteData.sso_enabled}
							<!-- IdP Metadata URL -->
							<Input
								id="invite-idp-metadata-url"
								name="idp_metadata_url"
								type="text"
								label="IdPメタデータURL"
								value={$inviteData.idp_metadata_url ?? ""}
								oninput={(e) => {
									const target = e.currentTarget as HTMLInputElement;
									$inviteData.idp_metadata_url = target.value;
								}}
								error={$inviteErrors.idp_metadata_url?.[0]}
								required
								placeholder="IdPメタデータURL。例）https://idp.example.com/metadata"
								variant="underlined"
							/>

							<!-- Allowed Domains -->
							<div class="space-y-3">
								<div class="block text-sm font-medium text-gray-700">許可ドメイン</div>
								{#each $inviteData.allowed_domains as domain, index (domain.key)}
									<div class="flex items-center gap-2">
										<Input
											id="allowed-domain-{index}"
											name="allowed_domains.{index}.value"
											type="text"
											label=""
											bind:value={$inviteData.allowed_domains[index].value}
											placeholder="example.com"
											variant="underlined"
										/>
										{#if $inviteData.allowed_domains.length > 1}
											<Button
												type="button"
												variant="secondary"
												onclick={() => removeAllowedDomain(index)}
												data-testid="remove-domain-{index}"
											>
												削除
											</Button>
										{/if}
									</div>
								{/each}
								<Button
									type="button"
									variant="secondary"
									onclick={addAllowedDomain}
									data-testid="add-domain-button"
								>
									+ ドメインを追加
								</Button>
								{#if $inviteErrors.allowed_domains?.[0]?.value}
									<p class="text-sm text-red-600">{$inviteErrors.allowed_domains[0].value}</p>
								{/if}
							</div>
						{/if}

						<!-- Generate token button (not primary button of slideover) -->
						<Button
							type="submit"
							variant="primary"
							loading={$inviteIsSubmitting}
							disabled={$inviteIsSubmitting || !isInviteFormValid()}
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

							{#each Object.entries(editingTenant.enterprise_features) as [featureKey] (featureKey)}
								{@const editable = isFeatureEditable(featureKey)}
								{@const isEnabled =
									$editData.enterprise_features[featureKey as keyof EnterpriseFeatures]?.enabled}

								<div class="border border-gray-200 rounded-lg p-4">
									<div class="flex items-center justify-between">
										<div class="flex items-center gap-2">
											<h4 class="text-sm font-medium text-gray-900">
												{featureKeyToLabelMap[featureKey]}
											</h4>
											{#if !editable}
												<span class="text-xs text-gray-500">(読み取り専用)</span>
											{/if}
										</div>
										<div class="flex gap-2">
											<InteractivePill
												selected={!isEnabled}
												onclick={editable
													? () =>
															($editData.enterprise_features[
																featureKey as keyof EnterpriseFeatures
															].enabled = false)
													: undefined}
												variant={editable ? "orange" : "gray"}
												class={!editable ? "opacity-50" : ""}
												data-testid={`${featureKey}-disabled`}
											>
												無効
											</InteractivePill>
											<InteractivePill
												selected={isEnabled}
												onclick={editable
													? () =>
															($editData.enterprise_features[
																featureKey as keyof EnterpriseFeatures
															].enabled = true)
													: undefined}
												variant={editable ? "orange" : "gray"}
												class={!editable ? "opacity-50" : ""}
												data-testid={`${featureKey}-enabled`}
											>
												有効
											</InteractivePill>
										</div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				</Slideover>
			</form>
		{/if}

		{#if loginTenant}
			<form
				use:loginForm
				class="space-y-6 p-1 flex flex-col h-full"
				data-testid="login-tenant-form"
			>
				<Slideover
					title={`${loginTenant.name}にログイン`}
					primaryButtonText="ログイン"
					primaryButtonTypeSubmit={true}
					onClose={handleLoginClose}
					loading={$loginIsSubmitting}
					data-testid="login-tenant-slideover"
				>
					<div class="flex-grow space-y-6">
						<Alert type="warning" title="注意" data-testid="login-security-alert">
							<p>この操作により、お客様のテナントにログインします。</p>
						</Alert>

						<div class="space-y-2">
							<p class="text-sm text-gray-600">
								<strong>ログイン先テナント:</strong>
								{loginTenant.name}
							</p>
							<p class="text-sm text-gray-500 font-mono">
								ID: {loginTenant.id}
							</p>
						</div>

						<div class="p-3 bg-blue-50 border border-blue-200 rounded-md">
							<p class="text-sm text-blue-800">上記のテナント名を正確に入力してください。</p>
						</div>

						<Input
							id="login-tenant-name"
							name="tenant_name"
							type="text"
							label="確認のためテナント名を入力してください"
							bind:value={$loginData.tenant_name}
							error={$loginErrors.tenant_name?.[0]}
							required
							placeholder={loginTenant.name}
							variant="underlined"
						/>
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
												class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium sm:pl-6"
												data-testid={`tenant-name-${tenant.id}`}
											>
												<a
													href={resolve(`/tenants/${tenant.id}/users`)}
													class="text-indigo-600 hover:text-indigo-900 font-medium"
													data-testid={`tenant-users-link-${tenant.id}`}
												>
													{tenant.name}
												</a>
											</td>
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
												<div class="flex items-center justify-end space-x-2">
													<Button
														type="button"
														variant="link"
														onclick={() => handleEditTenant(tenant)}
														data-testid={`edit-tenant-button-${tenant.id}`}
													>
														編集
													</Button>
													<Button
														type="button"
														variant="link"
														onclick={() => handleLoginOpen(tenant)}
														data-testid={`login-tenant-button-${tenant.id}`}
													>
														ログイン
													</Button>
												</div>
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
