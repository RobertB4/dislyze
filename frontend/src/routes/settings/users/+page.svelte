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
	import { invalidateAll } from "$app/navigation";
	import Badge from "$components/Badge.svelte";
	import Select from "$components/Select.svelte";
	import Alert from "$components/Alert.svelte";

	let { data: pageData }: { data: PageData } = $props();

	let isSlideoverOpen = $state(false);
	let userToDelete = $state<{ id: string; name: string; email: string } | null>(null);

	const { form, data, errors, isSubmitting, reset } = createForm({
		initialValues: {
			email: "",
			name: "",
			role: "editor"
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.email = values.email.trim();
			values.name = values.name.trim();

			if (!values.name) {
				errs.name = "氏名は必須です";
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

				if (responseData.error) {
					throw new KnownError(responseData.error);
				}

				toast.show("ユーザーを招待しました。", "success");
				isSlideoverOpen = false;
				reset();
				await invalidateAll();
			} catch (err) {
				console.log("catch", err);
				toast.showError(err);
			}
		}
	});

	const {
		form: deleteForm,
		data: deleteData,
		errors: deleteErrors,
		isSubmitting: isDeleting,
		reset: resetDelete
	} = createForm({
		initialValues: {
			confirmEmail: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.confirmEmail = values.confirmEmail.trim();

			if (!values.confirmEmail) {
				errs.confirmEmail = "メールアドレスの入力は必須です";
			} else if (values.confirmEmail !== userToDelete?.email) {
				errs.confirmEmail = "メールアドレスが一致しません";
			}
			return errs;
		},
		onSubmit: async () => {
			if (!userToDelete) return;

			try {
				const response = await fetch(`${PUBLIC_API_URL}/users/${userToDelete.id}`, {
					method: "DELETE",
					credentials: "include"
				});

				if (response.status === 204) {
					toast.show("ユーザーを削除しました。", "success");
					userToDelete = null;
					resetDelete();
					await invalidateAll();
					return;
				}

				const responseData: { error?: string } = await response.json();
				if (responseData.error) {
					throw new KnownError(responseData.error);
				}

				if (!response.ok) {
					throw new Error(
						`request to /users/${userToDelete.id} failed with status ${response.status}`
					);
				}
			} catch (err) {
				toast.showError(err);
			}
		}
	});

	const handleClose = () => {
		isSlideoverOpen = false;
		reset();
	};

	const handleDeleteModalClose = () => {
		userToDelete = null;
		resetDelete();
	};

	const handleResendInvite = async (userId: string) => {
		try {
			const response = await fetch(`${PUBLIC_API_URL}/users/${userId}/resend-invite`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				credentials: "include"
			});

			const responseData: { error?: string } = await response.json();
			if (responseData.error) {
				throw new KnownError(responseData.error);
			}

			if (!response.ok) {
				throw new Error(
					`request to /users/${userId}/resend-invite failed with status ${response.status}`
				);
			}

			toast.show("招待メールを送信しました。", "success");
		} catch (err) {
			toast.showError(err);
		}
	};

	const handleDeleteUser = (user: { id: string; name: string; email: string }) => {
		userToDelete = user;
	};

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

	const roleMap: Record<string, string> = {
		admin: "管理者",
		editor: "編集者"
	};
</script>

<Layout pageTitle="ユーザー管理">
	{#snippet buttons()}
		<Button type="button" variant="primary" onclick={() => (isSlideoverOpen = true)}>
			ユーザーを追加
		</Button>
	{/snippet}

	{#if isSlideoverOpen}
		<form use:form class="space-y-6 p-1 flex flex-col h-full">
			<Slideover
				title="ユーザーを追加"
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
						placeholder="メールアドレス"
						variant="underlined"
					/>
					<Input
						id="name"
						name="name"
						type="text"
						label="氏名"
						bind:value={$data.name}
						error={$errors.name?.[0]}
						required
						placeholder="氏名"
						variant="underlined"
					/>
					<Select
						id="role"
						name="role"
						label="役割"
						options={[
							{ value: "editor", label: "編集者" },
							{ value: "admin", label: "管理者" }
						]}
						bind:value={$data.role}
					/>
				</div>
			</Slideover>
		</form>
	{/if}

	{#if userToDelete}
		<form use:deleteForm class="space-y-6 p-1 flex flex-col h-full">
			<Slideover
				title="ユーザーを削除"
				primaryButtonText="削除"
				primaryButtonTypeSubmit={true}
				onClose={handleDeleteModalClose}
				loading={$isDeleting}
			>
				<div class="flex-grow space-y-6">
					<Alert type="danger" title="この操作は元に戻せません。">
						<p>
							削除を確認するには、ユーザーのメールアドレス<strong>「{userToDelete.email}」</strong
							>を入力してください。
						</p>
					</Alert>
					<Input
						id="confirmEmail"
						name="confirmEmail"
						type="email"
						label="メールアドレスを入力して確認"
						bind:value={$deleteData.confirmEmail}
						error={$deleteErrors.confirmEmail?.[0]}
						required
						placeholder={userToDelete.email}
						variant="underlined"
					/>
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
									>氏名</th
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
									<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
										><Badge color={statusMap[user.status].color}
											>{statusMap[user.status].label}</Badge
										>
										{#if user.status === "pending_verification"}
											<Button
												variant="link"
												class="ml-2 text-sm text-indigo-600 hover:text-indigo-900"
												onclick={() => handleResendInvite(user.id)}>招待メールを再送信</Button
											>
										{/if}
									</td>
									<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">{user.email}</td>
									<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
										>{roleMap[user.role]}</td
									>
									<td
										class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
									>
										{#if user.status === "pending_verification"}
											<Button
												variant="link"
												class="mr-4 text-sm text-red-600 hover:text-red-900"
												onclick={() => handleDeleteUser(user)}>招待をキャンセル</Button
											>
										{/if}
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
