<script lang="ts">
	import Button from "$components/Button.svelte";
	import Layout from "$components/Layout.svelte";
	import type { PageData } from "./$types";
	export let data: PageData;
</script>

<Layout>
	{#if data.error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4">
			<h2 class="text-red-800 font-semibold">エラー</h2>
			<p class="text-red-600">{data.error}</p>
		</div>
	{:else if data.users && data.users.length > 0}
		<div class="sm:flex sm:justify-end mt-6">
			<div class="mt-4 sm:mt-0 sm:flex-none">
				<Button type="button" variant="primary">ユーザーを追加</Button>
			</div>
		</div>
		<div class="mt-8 flow-root">
			<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
				<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
					<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
						<table class="min-w-full divide-y divide-gray-300">
							<thead class="bg-gray-50">
								<tr>
									<th
										scope="col"
										class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6"
										>名前</th
									>
									<th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
										>ステータス</th
									>
									<th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
										>メールアドレス</th
									>
									<th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
										>役割</th
									>
									<th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-6">
										<span class="sr-only">編集</span>
									</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-gray-200 bg-white">
								{#each data.users as user (user.id)}
									<tr>
										<td
											class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
											>{user.name}</td
										>
										<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">{user.status}</td>
										<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">{user.email}</td>
										<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">{user.role}</td>
										<td
											class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
										>
											<a href="#" class="text-indigo-600 hover:text-indigo-900"
												>編集<span class="sr-only">, {user.name}</span></a
											>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		</div>
	{:else}
		<div class="bg-white shadow rounded-lg p-6">
			<h2 class="text-lg font-medium text-gray-900">ユーザー</h2>
			<p class="mt-2 text-sm text-gray-500">ユーザーが見つかりませんでした。</p>
			<div class="mt-6">
				<button
					type="button"
					class="inline-flex items-center rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
				>
					<!-- Heroicon name: mini/plus -->
					<svg
						class="-ml-0.5 mr-1.5 h-5 w-5"
						viewBox="0 0 20 20"
						fill="currentColor"
						aria-hidden="true"
					>
						<path
							d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z"
						/>
					</svg>
					ユーザーを追加
				</button>
			</div>
		</div>
	{/if}
</Layout>
