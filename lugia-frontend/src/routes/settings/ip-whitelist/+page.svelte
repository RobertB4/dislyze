<script lang="ts">
	import { Button, Badge, toast } from "@dislyze/zoroark";
	import Layout from "$components/Layout.svelte";
	import SettingsTabs from "../SettingsTabs.svelte";
	import Skeleton from "./Skeleton.svelte";
	import AddIPModal from "./AddIPModal.svelte";
	import EditLabelModal from "./EditLabelModal.svelte";
	import DeleteConfirmModal from "./DeleteConfirmModal.svelte";
	import ActivationWarningModal from "./ActivationWarningModal.svelte";
	import DeactivationWarningModal from "./DeactivationWarningModal.svelte";
	import type { PageData } from "./$types";
	import { hasPermission } from "$lib/authz";
	import { mutationFetch } from "$lib/fetch";
	import { invalidate } from "$app/navigation";
	import { forceUpdateMeCache } from "@dislyze/zoroark";
	import type { IPWhitelistRule } from "./+page";

	let { data: pageData }: { data: PageData } = $props();

	let isAddIpSlideoverOpen = $state(false);
	let isActivationModalOpen = $state(false);
	let isDeactivationModalOpen = $state(false);
	let selectedWhitelistRule = $state<IPWhitelistRule | null>(null);
	let whitelistRuleToDelete = $state<IPWhitelistRule | null>(null);
	let warningUserIP = $state<string | null>(null);

	const handleActivate = async () => {
		const { success, response } = await mutationFetch(`/api/ip-whitelist/activate`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json"
			},
			body: JSON.stringify({ force: false })
		});

		if (success) {
			const contentType = response.headers.get("content-type");
			if (contentType && contentType.includes("application/json")) {
				const responseData = await response.json();

				if (!responseData.user_ip) {
					throw new Error(`Expected responseData.user_ip to be defined`);
				}

				warningUserIP = responseData.user_ip;
				isActivationModalOpen = true;
			} else {
				// Empty response - activation successful without warning
				forceUpdateMeCache.set(true);
				await invalidate((u) => u.pathname.includes("/api/me"));
				toast.show("IPアドレス制限を有効にしました", "success");
			}
		}
	};

	const handleDeactivate = () => {
		isDeactivationModalOpen = true;
	};
</script>

<Layout
	me={pageData.me}
	pageTitle="IPアドレス制限"
	promises={{
		ipWhitelistResponse: pageData.ipWhitelistPromise
	}}
>
	{#snippet buttons()}
		{#if hasPermission(pageData.me, "ip_whitelist.edit")}
			<Button
				type="button"
				variant="primary"
				onclick={() => (isAddIpSlideoverOpen = true)}
				data-testid="add-ip-button"
			>
				IPアドレスを追加
			</Button>
		{/if}
	{/snippet}

	{#snippet skeleton()}
		<Skeleton />
	{/snippet}

	{#snippet children({ ipWhitelistResponse })}
		{@const ipRules = ipWhitelistResponse}
		{@const isActive = pageData.me.enterprise_features.ip_whitelist.active}

		<SettingsTabs me={pageData.me} />

		<!-- Status Section -->
		<div class="mb-6 p-4 bg-gray-50 rounded-lg" data-testid="status-section">
			<div class="flex items-center justify-between">
				<div class="flex items-center space-x-3">
					<h3 class="text-lg font-medium text-gray-900">IPアドレス制限の状態</h3>
					<Badge color={isActive ? "green" : "yellow"} data-testid="status-badge">
						{isActive ? "有効" : "無効"}
					</Badge>
				</div>
				{#if hasPermission(pageData.me, "ip_whitelist.edit")}
					<Button
						type="button"
						variant={isActive ? "secondary" : "primary"}
						onclick={() => {
							if (isActive) {
								handleDeactivate();
							} else {
								handleActivate();
							}
						}}
						data-testid="toggle-activation-button"
					>
						{isActive ? "無効にする" : "有効にする"}
					</Button>
				{/if}
			</div>
			<p class="mt-2 text-sm text-gray-600">
				{#if isActive}
					IPアドレス制限が有効です。下記のIPアドレスからのみアクセスが許可されます。
				{:else}
					IPアドレス制限が無効です。すべてのIPアドレスからアクセスが許可されます。
				{/if}
			</p>
		</div>

		<!-- IP Whitelist Table -->
		<div class="mt-8 flow-root">
			{#if ipRules.length === 0}
				<div class="text-center py-12" data-testid="no-ip-rules-message">
					<div class="text-gray-500 text-lg">IPアドレスが登録されていません</div>
					<p class="text-gray-400 text-sm mt-2">「IPアドレスを追加」ボタンから設定できます</p>
				</div>
			{:else}
				<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
					<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
						<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
							<table class="min-w-full divide-y divide-gray-300" data-testid="ip-whitelist-table">
								<thead class="bg-gray-50" data-testid="ip-whitelist-table-header">
									<tr>
										<th
											scope="col"
											class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6"
											data-testid="ip-table-header-address"
										>
											IPアドレス/CIDR
										</th>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="ip-table-header-label"
										>
											説明
										</th>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="ip-table-header-created-at"
										>
											作成日時
										</th>
										<th
											scope="col"
											class="relative py-3.5 pl-3 pr-4 sm:pr-6"
											data-testid="ip-table-header-actions"
										>
											<span class="sr-only">操作</span>
										</th>
									</tr>
								</thead>
								<tbody
									class="divide-y divide-gray-200 bg-white"
									data-testid="ip-whitelist-table-body"
								>
									{#each ipRules as rule (rule.id)}
										<tr data-testid={`ip-rule-row-${rule.id}`} class="hover:bg-gray-50">
											<td
												class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
												data-testid={`ip-address-${rule.id}`}
											>
												<code class="text-sm bg-gray-100 px-2 py-1 rounded">{rule.ip_address}</code>
											</td>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
												data-testid={`ip-label-${rule.id}`}
											>
												{rule.label}
											</td>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
												data-testid={`ip-created-at-${rule.id}`}
											>
												{new Date(rule.created_at).toLocaleDateString("ja-JP", {
													year: "numeric",
													month: "2-digit",
													day: "2-digit"
												})}
											</td>
											<td
												class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
												data-testid={`ip-actions-${rule.id}`}
											>
												{#if hasPermission(pageData.me, "ip_whitelist.edit")}
													<Button
														variant="link"
														class="mr-4 text-sm text-indigo-600 hover:text-indigo-900"
														onclick={() => (selectedWhitelistRule = rule)}
														data-testid={`edit-label-button-${rule.id}`}
													>
														編集
													</Button>
													<Button
														variant="link"
														class="text-sm text-red-600 hover:text-red-900"
														onclick={() => {
															whitelistRuleToDelete = rule;
														}}
														data-testid={`delete-ip-button-${rule.id}`}
													>
														削除
													</Button>
												{/if}
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

		{#if isAddIpSlideoverOpen}
			<AddIPModal onClose={() => (isAddIpSlideoverOpen = false)} existingRules={ipRules} />
		{/if}

		{#if selectedWhitelistRule !== null}
			<EditLabelModal onClose={() => (selectedWhitelistRule = null)} rule={selectedWhitelistRule} />
		{/if}

		{#if whitelistRuleToDelete !== null}
			<DeleteConfirmModal
				onClose={() => (whitelistRuleToDelete = null)}
				rule={whitelistRuleToDelete}
			/>
		{/if}

		{#if isActivationModalOpen && warningUserIP}
			<ActivationWarningModal
				onClose={() => {
					isActivationModalOpen = false;
					warningUserIP = null;
				}}
				userIP={warningUserIP}
			/>
		{/if}

		{#if isDeactivationModalOpen}
			<DeactivationWarningModal onClose={() => (isDeactivationModalOpen = false)} />
		{/if}
	{/snippet}
</Layout>
