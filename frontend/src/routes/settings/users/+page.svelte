<script lang="ts">
	import Button from "$components/Button.svelte";
	import Layout from "$components/Layout.svelte";
	import Slideover from "$components/Slideover.svelte";
	import type { PageData } from "./$types";
	import { createForm } from "felte";
	import Input from "$components/Input.svelte";
	import { toast } from "$components/Toast/toast";
	import { PUBLIC_API_URL } from "$env/static/public";
	import { KnownError } from "$lib/errors";

	let { data: pageData }: { data: PageData } = $props();

	let isSlideoverOpen = $state(false);

	const { form, data, errors, isSubmitting, reset } = createForm({
		initialValues: {
			email: "",
			name: "",
			role: "user"
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.email = values.email.trim();
			values.name = values.name.trim();

			if (!values.name) {
				errs.name = "名前は必須です";
			}
			if (!values.email) {
				errs.email = "メールアドレスは必須です";
			} else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
				errs.email = "メールアドレスの形式が正しくありません";
			}
			return errs;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`${PUBLIC_API_URL}/users/invite`, {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify(values),
					credentials: "include"
				});

				const responseData: { error?: string } = await response.json();

				console.log({ responseData });

				if (responseData.error) {
					throw new KnownError(responseData.error || "招待の送信に失敗しました。");
				}

				toast.show("ユーザーを招待しました。", "success");
				isSlideoverOpen = false;
				reset();
			} catch (err) {
				console.log("catch", err);
				toast.showError(err);
			}
		}
	});

	const handleClose = () => {
		isSlideoverOpen = false;
		reset();
	};
</script>

<Layout pageTitle="ユーザー管理">
	{#snippet buttons()}
		<Button type="button" variant="primary" on:click={() => (isSlideoverOpen = true)}>
			ユーザーを追加
		</Button>
	{/snippet}

	{#if isSlideoverOpen}
		<form use:form class="space-y-6 p-1 flex flex-col h-full">
			<Slideover
				title="ユーザーを追加"
				subtitle="新しいユーザーを招待します"
				primaryButtonText="招待を送信"
				primaryButtonTypeSubmit={true}
				onClose={handleClose}
				loading={$isSubmitting}
			>
				<div class="flex-grow space-y-6">
					<Input
						id="email"
						name="email"
						type="email"
						label="メールアドレス"
						bind:value={$data.email}
						error={$errors.email?.[0]}
						required
						placeholder="user@example.com"
					/>
					<Input
						id="name"
						name="name"
						type="text"
						label="名前"
						bind:value={$data.name}
						error={$errors.name?.[0]}
						required
						placeholder="山田 太郎"
					/>
					<div>
						<label for="role" class="block text-sm font-medium text-gray-700">役割</label>
						<select
							id="role"
							name="role"
							bind:value={$data.role}
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
						>
							<option value="user">一般ユーザー</option>
							<option value="admin">管理者</option>
						</select>
					</div>
				</div>
			</Slideover>
		</form>
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
							{#each pageData.users as user (user.id)}
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
