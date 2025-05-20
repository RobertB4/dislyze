<script lang="ts">
	import Button from "$components/Button.svelte";
	import Layout from "$components/Layout.svelte";
	import Slideover from "$components/Slideover.svelte";
	import type { PageData } from "./$types";

	let { data } = $props<{ data: PageData }>();

	let isSlideoverOpen = $state(false);
	let formData = $state({
		email: "",
		name: "",
		role: "user"
	});

	const handleClose = () => {
		isSlideoverOpen = false;
	};
</script>

<Layout>
	<div slot="buttons">
		<Button type="button" variant="primary" on:click={() => (isSlideoverOpen = true)}
			>ユーザーを追加</Button
		>
	</div>

	{#if isSlideoverOpen}
		<Slideover
			title="ユーザーを追加"
			subtitle="新しいユーザーを招待します"
			primaryButtonText="招待を送信"
			onClose={handleClose}
		>
			<div class="space-y-4">
				<div>
					<label for="email" class="block text-sm font-medium text-gray-700">メールアドレス</label>
					<input
						type="email"
						name="email"
						id="email"
						bind:value={formData.email}
						class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
						placeholder="user@example.com"
					/>
				</div>
				<div>
					<label for="name" class="block text-sm font-medium text-gray-700">名前</label>
					<input
						type="text"
						name="name"
						id="name"
						bind:value={formData.name}
						class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
						placeholder="山田 太郎"
					/>
				</div>
				<div>
					<label for="role" class="block text-sm font-medium text-gray-700">役割</label>
					<select
						id="role"
						name="role"
						bind:value={formData.role}
						class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
					>
						<option value="user">一般ユーザー</option>
						<option value="admin">管理者</option>
					</select>
				</div>
			</div>
		</Slideover>
	{/if}

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
</Layout>
