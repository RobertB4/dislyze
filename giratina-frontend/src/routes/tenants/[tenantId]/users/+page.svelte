<script lang="ts">
	import { Badge } from "@dislyze/zoroark";
	import Layout from "$components/Layout.svelte";
	import type { PageData } from "./$types";

	let { data: pageData }: { data: PageData } = $props();

	const statusMap: Record<string, { label: string; color: "green" | "yellow" | "red" }> = {
		active: {
			label: "有効",
			color: "green"
		},
		pending_verification: {
			label: "招待済み",
			color: "yellow"
		},
		suspended: {
			label: "停止中",
			color: "red"
		}
	};
</script>

<Layout
	me={pageData.me}
	pageTitle="ユーザー一覧"
	promises={{
		usersResponse: pageData.usersPromise
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

	{#snippet children({ usersResponse })}
		{@const { users } = usersResponse}

		<div class="mt-8 flow-root">
			{#if users.length === 0}
				<div class="text-center py-12" data-testid="no-users-message">
					<div class="text-gray-500 text-lg">ユーザーが見つかりませんでした</div>
				</div>
			{:else}
				<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
					<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
						<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
							<table class="min-w-full divide-y divide-gray-300" data-testid="users-table">
								<thead class="bg-gray-50" data-testid="users-table-header">
									<tr>
										<th
											scope="col"
											class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6"
											data-testid="user-table-header-name"
										>
											氏名
										</th>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="user-table-header-email"
										>
											メールアドレス
										</th>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="user-table-header-status"
										>
											ステータス
										</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-200 bg-white" data-testid="users-table-body">
									{#each users as user (user.id)}
										<tr data-testid={`user-row-${user.id}`}>
											<td
												class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
												data-testid={`user-name-${user.id}`}
											>
												{user.name}
											</td>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
												data-testid={`user-email-${user.id}`}
											>
												{user.email}
											</td>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
												data-testid={`user-status-${user.id}`}
											>
												<Badge
													color={statusMap[user.status].color}
													data-testid={`user-status-badge-${user.id}`}
												>
													{statusMap[user.status].label}
												</Badge>
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
