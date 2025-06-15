<script lang="ts">
	import Layout from "$components/Layout.svelte";
	import { Badge, Tooltip } from "@dislyze/zoroark";
	import type { PageData } from "./$types";

	let { data: pageData }: { data: PageData } = $props();

	const featureKeyToLabelMap: Record<string, string> = {
		rbac: "権限設定"
	};
</script>

<Layout
	me={pageData.me}
	pageTitle="テナント一覧"
	promises={{
		tenantsResponse: pageData.tenantsPromise
	}}
>
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
