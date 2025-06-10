<script lang="ts">
	import Button from "$components/Button.svelte";
	import Layout from "$components/Layout.svelte";
	import SettingsTabs from "../SettingsTabs.svelte";
	import Skeleton from "./Skeleton.svelte";
	import Tooltip from "$components/Tooltip.svelte";
	import type { PageData } from "./$types";
	import type { RoleInfo } from "./+page";
	import { hasPermission } from "$lib/meCache";

	let { data: pageData }: { data: PageData } = $props();

	function sortRoles(roles: RoleInfo[]): RoleInfo[] {
		const defaultRoleOrder = ["管理者", "編集者", "閲覧者"];

		return roles.sort((a, b) => {
			if (a.is_default && b.is_default) {
				const aIndex = defaultRoleOrder.indexOf(a.name);
				const bIndex = defaultRoleOrder.indexOf(b.name);
				return aIndex - bIndex;
			}
			if (a.is_default) return -1;
			if (b.is_default) return 1;
			return a.name.localeCompare(b.name);
		});
	}
</script>

<Layout
	me={pageData.me}
	pageTitle="ロール管理"
	promises={{
		rolesResponse: pageData.rolesPromise
	}}
>
	{#snippet buttons()}
		{#if hasPermission(pageData.me, "roles.edit")}
			<Button
				type="button"
				variant="primary"
				onclick={() => {
					/* TODO: Open create role modal */
				}}
				data-testid="add-role-button"
			>
				ロールを追加
			</Button>
		{/if}
	{/snippet}

	{#snippet skeleton()}
		<Skeleton />
	{/snippet}

	{#snippet children({ rolesResponse })}
		{@const { roles } = rolesResponse}
		{@const sortedRoles = sortRoles(roles)}

		<SettingsTabs me={pageData.me} />

		<div class="mt-8 flow-root">
			{#if sortedRoles.length === 0}
				<div class="text-center py-12" data-testid="no-roles-message">
					<div class="text-gray-500 text-lg">ロールが見つかりませんでした</div>
				</div>
			{:else}
				<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
					<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
						<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
							<table class="min-w-full divide-y divide-gray-300" data-testid="roles-table">
								<thead class="bg-gray-50" data-testid="roles-table-header">
									<tr>
										<th
											scope="col"
											class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6"
											data-testid="role-table-header-name"
										>
											ロール名
										</th>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="role-table-header-description"
										>
											説明
										</th>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="role-table-header-permissions"
										>
											権限
										</th>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="role-table-header-type"
										>
											種類
										</th>
										<th
											scope="col"
											class="relative py-3.5 pl-3 pr-4 sm:pr-6"
											data-testid="role-table-header-actions"
										>
											<span class="sr-only">操作</span>
										</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-200 bg-white" data-testid="roles-table-body">
									{#each sortedRoles as role (role.id)}
										<tr data-testid={`role-row-${role.id}`}>
											<td
												class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
												data-testid={`role-name-${role.id}`}
											>
												{role.name}
											</td>
											<td
												class="px-3 py-4 text-sm text-gray-500"
												data-testid={`role-description-${role.id}`}
											>
												{role.description || "-"}
											</td>
											<td
												class="px-3 py-4 text-sm text-gray-500"
												data-testid={`role-permissions-${role.id}`}
											>
												{#if role.permissions.length === 0}
													<span class="text-gray-400">権限なし</span>
												{:else}
													<div class="flex flex-wrap gap-1 items-center">
														{#each role.permissions.slice(0, 3) as permission (permission)}
															<span
																class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-100 text-blue-800"
															>
																{permission}
															</span>
														{/each}
														{#if role.permissions.length > 3}
															<Tooltip class="ml-2">
																{#snippet content()}
																	<div class="space-y-1">
																		{#each role.permissions.slice(3) as permission (permission)}
																			<div class="text-xs">{permission}</div>
																		{/each}
																	</div>
																{/snippet}

																<span
																	class="text-gray-400 cursor-help border-b border-dotted border-gray-300 flex items-center"
																>
																	他{role.permissions.length - 3}件
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
												class="whitespace-nowrap px-3 py-4 text-sm"
												data-testid={`role-type-${role.id}`}
											>
												{#if role.is_default}
													<span
														class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-gray-100 text-gray-800"
													>
														デフォルト
													</span>
												{:else}
													<span
														class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-green-100 text-green-800"
													>
														カスタム
													</span>
												{/if}
											</td>
											<td
												class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
												data-testid={`role-actions-${role.id}`}
											>
												{#if hasPermission(pageData.me, "roles.edit") && !role.is_default}
													<Button
														variant="link"
														class="mr-4 text-sm text-red-600 hover:text-red-900"
														onclick={() => {
															/* TODO: Open delete modal */
														}}
														data-testid={`delete-role-button-${role.id}`}
													>
														削除
													</Button>
													<Button
														variant="link"
														class="text-indigo-600 hover:text-indigo-900"
														onclick={() => {
															/* TODO: Open edit modal */
														}}
														data-testid={`edit-role-button-${role.id}`}
													>
														編集
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
	{/snippet}
</Layout>
