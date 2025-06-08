<script lang="ts">
	import Button from "$components/Button.svelte";
	import Layout from "$components/Layout.svelte";
	import SettingsTabs from "../SettingsTabs.svelte";
	import Slideover from "$components/Slideover.svelte";
	import type { PageData } from "./$types";
	import { createForm } from "felte";
	import Input from "$components/Input.svelte";
	import { toast } from "$components/Toast/toast";
	import { invalidate } from "$app/navigation";
	import Badge from "$components/Badge.svelte";
	import Alert from "$components/Alert.svelte";
	import { mutationFetch } from "$lib/fetch";
	import Skeleton from "./Skeleton.svelte";
	import type { User } from "./+page";
	import { hasPermission } from "$lib/meCache";
	import { goto } from "$app/navigation";
	import Spinner from "$components/Spinner.svelte";
	import Tooltip from "$components/Tooltip.svelte";
	import RoleCard from "./RoleCard.svelte";

	let { data: pageData }: { data: PageData } = $props();

	let isSlideoverOpen = $state(false);
	let userToDelete = $state<{ id: string; name: string; email: string } | null>(null);
	let userToEdit = $state<User | null>(null);

	let searchTimeout: number | undefined;
	let isSearching = $state(false);

	function updateURL(
		page: number = pageData.currentPage,
		limit: number = pageData.currentLimit,
		search: string = pageData.currentSearch
	) {
		const params = new URLSearchParams();
		params.set("page", page.toString());
		params.set("limit", limit.toString());
		if (search) {
			params.set("search", search);
		}
		goto(`?${params.toString()}`, {
			replaceState: false,
			invalidate: [
				(url: URL) => {
					return url.pathname === "/api.users";
				}
			]
		});
	}

	function handleSearchInput(event: Event) {
		const target = event.target as HTMLInputElement;
		const inputValue = target.value;

		if (searchTimeout) {
			clearTimeout(searchTimeout);
		}

		isSearching = true;
		searchTimeout = window.setTimeout(() => {
			updateURL(1, pageData.currentLimit, inputValue); // Reset to page 1 when searching
			isSearching = false;
		}, 300);
	}

	function goToPage(page: number) {
		updateURL(page, pageData.currentLimit, pageData.currentSearch);
	}

	function goToFirstPage() {
		updateURL(1, pageData.currentLimit, pageData.currentSearch);
	}

	function goToLastPage(totalPages: number) {
		updateURL(totalPages, pageData.currentLimit, pageData.currentSearch);
	}

	const { form, data, errors, isSubmitting, reset } = createForm({
		initialValues: {
			email: "",
			name: "",
			roleIds: [] as string[]
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
			if (!values.roleIds || values.roleIds.length === 0) {
				errs.roleIds = "ロールを選択してください";
			}
			return errs;
		},
		onSubmit: async (values) => {
			const { success } = await mutationFetch(`/api/users/invite`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					email: values.email,
					name: values.name,
					role_ids: values.roleIds
				})
			});

			if (success) {
				await invalidate((u) => u.pathname === "/api/users");
				reset();
				toast.show("ユーザーを招待しました。", "success");
				isSlideoverOpen = false;
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

			const { success } = await mutationFetch(`/api/users/${userToDelete.id}/delete`, {
				method: "POST"
			});

			if (success) {
				await invalidate((u) => u.pathname === "/api/users");
				resetDelete();
				toast.show("ユーザーを削除しました。", "success");
				userToDelete = null;
			}
		}
	});

	const {
		form: editForm,
		data: editFormData,
		errors: editErrors,
		isSubmitting: isEditing,
		setInitialValues: setEditFormInitialValues,
		reset: resetEditForm
	} = createForm({
		initialValues: {
			roleIds: [] as string[]
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			if (!values.roleIds || values.roleIds.length === 0) {
				errs.roleIds = "ロールを選択してください";
			}
			return errs;
		},
		onSubmit: async (values) => {
			if (!userToEdit) return;

			const { success } = await mutationFetch(`/api/users/${userToEdit.id}/roles`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({ role_ids: values.roleIds })
			});

			if (success) {
				await invalidate((u) => u.pathname === "/api/users");
				toast.show("ユーザーのロールを更新しました。", "success");
				userToEdit = null;
				resetEditForm();
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

	const handleEditModalOpen = (user: User) => {
		setEditFormInitialValues({ roleIds: user.roles.map((role) => role.id) });
		userToEdit = user;
	};

	const handleEditModalClose = () => {
		userToEdit = null;
		resetEditForm();
	};

	const handleResendInvite = async (userId: string) => {
		const { success } = await mutationFetch(`/api/users/${userId}/resend-invite`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json"
			}
		});

		if (success) {
			toast.show("招待メールを送信しました。", "success");
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

	function isRoleSelected(roleId: string, selectedRoleIds: string[]): boolean {
		return selectedRoleIds.includes(roleId);
	}

	function toggleRole(roleId: string, currentRoleIds: string[]): string[] {
		if (currentRoleIds.includes(roleId)) {
			return currentRoleIds.filter((id) => id !== roleId);
		} else {
			return [...currentRoleIds, roleId];
		}
	}
</script>

<Layout
	me={pageData.me}
	pageTitle="ユーザー管理"
	promises={{
		usersResponse: pageData.usersPromise,
		rolesResponse: pageData.rolesPromise
	}}
>
	{#snippet buttons()}
		{#if hasPermission(pageData.me, "users.create")}
			<Button
				type="button"
				variant="primary"
				onclick={() => (isSlideoverOpen = true)}
				data-testid="add-user-button"
			>
				ユーザーを追加
			</Button>
		{/if}
	{/snippet}

	{#snippet skeleton()}
		<Skeleton />
	{/snippet}

	{#snippet children({ usersResponse, rolesResponse })}
		{@const { users, pagination } = usersResponse}
		{@const { roles } = rolesResponse}

		<SettingsTabs me={pageData.me} />

		<!-- Search bar -->
		<div class="mb-6" data-testid="search-section">
			<div class="max-w-md">
				<Input
					id="user-search"
					name="search"
					type="text"
					label="ユーザーを検索"
					placeholder="名前またはメールアドレスで検索"
					value={pageData.currentSearch}
					oninput={handleSearchInput}
					class="block w-full"
				/>
				{#if isSearching}
					<div class="mt-2 flex items-center text-sm text-gray-500" data-testid="search-loading">
						<Spinner />
						検索中...
					</div>
				{/if}
			</div>
		</div>
		{#if isSlideoverOpen}
			<form use:form class="space-y-6 p-1 flex flex-col h-full" data-testid="add-user-form">
				<Slideover
					title="ユーザーを追加"
					primaryButtonText="招待を送信"
					primaryButtonTypeSubmit={true}
					onClose={handleClose}
					loading={$isSubmitting}
					data-testid="add-user-slideover"
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
						<div class="space-y-3">
							<div class="block text-sm font-medium text-gray-700">ロール (複数選択可)</div>
							{#if $errors.roleIds?.[0]}
								<div class="text-sm text-red-600">{$errors.roleIds[0]}</div>
							{/if}
							<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
								{#each roles as role (role.id)}
									<RoleCard
										{role}
										isSelected={isRoleSelected(role.id, $data.roleIds)}
										onclick={() => {
											$data.roleIds = toggleRole(role.id, $data.roleIds);
										}}
										data-testid={`role-card-${role.id}`}
									/>
								{/each}
							</div>
						</div>
					</div>
				</Slideover>
			</form>
		{/if}

		{#if userToDelete}
			<form
				use:deleteForm
				class="space-y-6 p-1 flex flex-col h-full"
				data-testid="delete-user-form"
			>
				<Slideover
					title="ユーザーを削除"
					primaryButtonText="削除"
					primaryButtonTypeSubmit={true}
					onClose={handleDeleteModalClose}
					loading={$isDeleting}
					data-testid="delete-user-slideover"
				>
					<div class="flex-grow space-y-6">
						<Alert
							type="danger"
							title="この操作は元に戻せません。"
							data-testid="delete-user-warning"
						>
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

		{#if userToEdit}
			<form use:editForm class="space-y-6 p-1 flex flex-col h-full" data-testid="edit-user-form">
				<Slideover
					title="ユーザー権限を編集"
					primaryButtonText="保存"
					primaryButtonTypeSubmit={true}
					onClose={handleEditModalClose}
					loading={$isEditing}
					data-testid="edit-user-slideover"
				>
					<div class="flex-grow space-y-6">
						<p data-testid="edit-user-title">
							<strong>{userToEdit.name}</strong> ({userToEdit.email}) のロールを編集
						</p>
						<div class="space-y-3">
							<div class="block text-sm font-medium text-gray-700">ロール (複数選択可)</div>
							{#if $editErrors.roleIds?.[0]}
								<div class="text-sm text-red-600">{$editErrors.roleIds[0]}</div>
							{/if}
							<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
								{#each roles as role (role.id)}
									<RoleCard
										{role}
										isSelected={isRoleSelected(role.id, $editFormData.roleIds)}
										onclick={() => {
											$editFormData.roleIds = toggleRole(role.id, $editFormData.roleIds);
										}}
										data-testid={`edit-role-card-${role.id}`}
									/>
								{/each}
							</div>
						</div>
					</div>
				</Slideover>
			</form>
		{/if}

		<div class="mt-8 flow-root">
			{#if users.length === 0}
				<div class="text-center py-12" data-testid="no-users-message">
					<div class="text-gray-500 text-lg">
						{#if pageData.currentSearch}
							検索結果が見つかりませんでした
						{:else}
							ユーザーが見つかりませんでした
						{/if}
					</div>
					{#if pageData.currentSearch}
						<p class="text-gray-400 text-sm mt-2" data-testid="no-search-results-message">
							「{pageData.currentSearch}」に一致するユーザーはありません
						</p>
					{/if}
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
											data-testid="user-table-header-name">氏名</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="user-table-header-status">ステータス</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="user-table-header-email">メールアドレス</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											data-testid="user-table-header-role">ロール</th
										>
										<th
											scope="col"
											class="relative py-3.5 pl-3 pr-4 sm:pr-6"
											data-testid="user-table-header-actions"
										>
											<span class="sr-only">編集</span>
										</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-200 bg-white" data-testid="users-table-body">
									{#each users as user (user.id)}
										<tr data-testid={`user-row-${user.id}`}>
											<td
												class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
												data-testid={`user-name-${user.id}`}>{user.name}</td
											>
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
												{#if user.status === "pending_verification"}
													<Button
														variant="link"
														class="ml-2 text-sm text-indigo-600 hover:text-indigo-900"
														onclick={() => handleResendInvite(user.id)}
														data-testid={`resend-invite-button-${user.id}`}
													>
														招待メールを再送信
													</Button>
												{/if}
											</td>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
												data-testid={`user-email-${user.id}`}
											>
												{user.email}
											</td>
											<td
												class="whitespace-nowrap px-3 py-4 text-sm text-gray-500"
												data-testid={`user-role-${user.id}`}
											>
												{#if user.roles.length === 0}
													<span class="text-gray-400">ロールなし</span>
												{:else if user.roles.length === 1}
													{user.roles[0].name}
												{:else}
													<span>
														{user.roles[0].name}
														<Tooltip class="ml-2">
															{#snippet content()}
																{user.roles
																	.slice(1)
																	.map((role: { name: string }) => role.name)
																	.join("、")}
															{/snippet}

															<span
																class="text-gray-400 cursor-help border-b border-dotted border-gray-300 flex items-center"
															>
																他{user.roles.length - 1}件
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
													</span>
												{/if}
											</td>
											<td
												class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
												data-testid={`user-actions-${user.id}`}
											>
												{#if pageData.me.user_id !== user.id}
													{#if hasPermission(pageData.me, "users.delete")}
														{#if user.status === "pending_verification"}
															<Button
																variant="link"
																class="mr-4 text-sm text-red-600 hover:text-red-900"
																onclick={() => handleDeleteUser(user)}
																data-testid={`cancel-invite-button-${user.id}`}
															>
																招待をキャンセル
															</Button>
														{:else}
															<Button
																variant="link"
																class="mr-4 text-sm text-red-600 hover:text-red-900"
																onclick={() => handleDeleteUser(user)}
																data-testid={`delete-user-button-${user.id}`}
															>
																削除
															</Button>
														{/if}
													{/if}
													{#if hasPermission(pageData.me, "users.update")}
														<Button
															variant="link"
															class="text-indigo-600 hover:text-indigo-900"
															onclick={() => handleEditModalOpen(user)}
															data-testid={`edit-permissions-button-${user.id}`}
														>
															権限編集<span class="sr-only">, {user.name}</span>
														</Button>
													{/if}
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

		<!-- Pagination controls -->
		{#if pagination.total > 0}
			<div class="mt-6 flex items-center justify-between" data-testid="pagination-controls">
				<div class="text-sm text-gray-700" data-testid="pagination-info">
					{pagination.total}件中 {Math.min(
						(pagination.page - 1) * pagination.limit + 1,
						pagination.total
					)} - {Math.min(pagination.page * pagination.limit, pagination.total)}件を表示
				</div>
				<div class="flex items-center space-x-2" data-testid="pagination-buttons">
					<!-- First page button -->
					<Button
						variant="secondary"
						onclick={goToFirstPage}
						disabled={!pagination.has_prev}
						class="px-3 py-1.5 text-sm"
						data-testid="pagination-first"
					>
						«
					</Button>

					<!-- Previous page button -->
					<Button
						variant="secondary"
						onclick={() => goToPage(pagination.page - 1)}
						disabled={!pagination.has_prev}
						class="px-3 py-1.5 text-sm"
						data-testid="pagination-prev"
					>
						‹
					</Button>

					<!-- Current page info -->
					<span class="px-3 py-1.5 text-sm text-gray-600" data-testid="pagination-current">
						{pagination.page} / {pagination.total_pages}
					</span>

					<!-- Next page button -->
					<Button
						variant="secondary"
						onclick={() => goToPage(pagination.page + 1)}
						disabled={!pagination.has_next}
						class="px-3 py-1.5 text-sm"
						data-testid="pagination-next"
					>
						›
					</Button>

					<!-- Last page button -->
					<Button
						variant="secondary"
						onclick={() => goToLastPage(pagination.total_pages)}
						disabled={!pagination.has_next}
						class="px-3 py-1.5 text-sm"
						data-testid="pagination-last"
					>
						»
					</Button>
				</div>
			</div>
		{/if}
	{/snippet}
</Layout>
