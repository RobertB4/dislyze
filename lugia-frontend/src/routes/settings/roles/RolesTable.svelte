<script lang="ts">
	import { Button, Tooltip, Slideover, Input, Alert, Badge, toast } from "@dislyze/zoroark";
	import SettingsTabs from "../SettingsTabs.svelte";
	import PermissionSelector from "./PermissionSelector.svelte";
	import type { PermissionInfo, RoleInfo } from "./+page";
	import { type Me } from "@dislyze/zoroark";
	import { hasPermission } from "$lib/authz";
	import { createForm } from "felte";
	import { invalidate } from "$app/navigation";
	import { mutationFetch } from "$lib/fetch";

	let {
		me,
		roles,
		permissions,
		isCreateSlideoverOpen = $bindable()
	}: {
		me: Me;
		roles: RoleInfo[];
		permissions: PermissionInfo[];
		isCreateSlideoverOpen: boolean;
	} = $props();

	let editingRole = $state<RoleInfo | null>(null);
	let roleToDelete = $state<RoleInfo | null>(null);

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

	const { form, data, errors, isSubmitting, reset, setFields } = createForm({
		initialValues: {
			name: "",
			description: "",
			permission_ids: [] as string[],
			hasPermission: null
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.name = values.name.trim();
			values.description = values.description.trim();

			if (!values.name) {
				errs.name = "ロール名は必須です";
			} else if (roles.some(role => role.name === values.name)) {
				errs.name = "このロール名は既に使用されています";
			}

			if (values.permission_ids.length === 0) {
				errs.hasPermission = "権限を選択してください。";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const { success } = await mutationFetch(`/api/roles/create`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					name: values.name,
					description: values.description,
					permission_ids: values.permission_ids
				})
			});

			if (success) {
				await invalidate((u) => u.pathname === "/api/roles");
				reset();
				toast.show("ロールを作成しました。", "success");
				isCreateSlideoverOpen = false;
			}
		}
	});

	const handleCreateClose = () => {
		isCreateSlideoverOpen = false;
		reset();
	};

	const {
		form: editForm,
		data: editData,
		errors: editErrors,
		isSubmitting: editIsSubmitting,
		reset: editReset,
		setInitialValues: setEditFormInitialValues,
		setFields: editSetFields
	} = createForm({
		initialValues: {
			name: "",
			description: "",
			permission_ids: [] as string[],
			hasPermission: null
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.name = values.name.trim();
			values.description = values.description.trim();

			if (!values.name) {
				errs.name = "ロール名は必須です。";
			}

			if (values.permission_ids.length === 0) {
				errs.hasPermission = "権限を選択してください。";
			}

			return errs;
		},
		onSubmit: async (values) => {
			if (!editingRole) return;

			const { success } = await mutationFetch(`/api/roles/${editingRole.id}/update`, {
				method: "POST",
				body: JSON.stringify({
					name: values.name,
					description: values.description,
					permission_ids: values.permission_ids
				})
			});

			if (success) {
				await invalidate((u) => u.pathname === "/api/roles");
				editReset();
				toast.show("ロールを更新しました。", "success");
				editingRole = null;
			}
		}
	});

	const handleEditModalOpen = (role: RoleInfo) => {
		const rolePermissionIds = role.permissions.map((p) => p.id);
		setEditFormInitialValues({
			name: role.name,
			description: role.description,
			permission_ids: rolePermissionIds,
			hasPermission: null
		});
		editingRole = role;
	};

	const handleEditClose = () => {
		editingRole = null;
		editReset();
	};

	const handleDeleteRole = (role: RoleInfo) => {
		roleToDelete = role;
	};

	const handleDeleteModalClose = () => {
		roleToDelete = null;
		deleteReset();
	};

	const {
		form: deleteForm,
		data: deleteData,
		errors: deleteErrors,
		isSubmitting: isDeleting,
		reset: deleteReset
	} = createForm({
		initialValues: {
			confirmName: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.confirmName = values.confirmName.trim();

			if (values.confirmName !== roleToDelete?.name) {
				errs.confirmName = "ロール名が一致しません";
			}
			return errs;
		},
		onSubmit: async () => {
			if (!roleToDelete) return;

			const { success } = await mutationFetch(`/api/roles/${roleToDelete.id}/delete`, {
				method: "POST"
			});

			if (success) {
				await invalidate((u) => u.pathname === "/api/roles");
				deleteReset();
				toast.show("ロールを削除しました。", "success");
				roleToDelete = null;
			}
		}
	});

	let sortedRoles = $derived(sortRoles(roles));
</script>

{#if isCreateSlideoverOpen}
	<form use:form class="space-y-6 p-1 flex flex-col h-full" data-testid="create-role-form">
		<Slideover
			title="ロールを作成"
			primaryButtonText="作成"
			primaryButtonTypeSubmit={true}
			onClose={handleCreateClose}
			loading={$isSubmitting}
			data-testid="create-role-slideover"
		>
			<div class="flex-grow space-y-6">
				<Input
					id="name"
					name="name"
					type="text"
					label="ロール名"
					bind:value={$data.name}
					error={$errors.name?.[0]}
					required
					placeholder="例: カスタムロール"
					variant="underlined"
				/>
				<Input
					id="description"
					name="description"
					type="text"
					label="説明"
					bind:value={$data.description}
					error={$errors.description?.[0]}
					placeholder="ロールの説明（任意）"
					variant="underlined"
				/>
				<PermissionSelector
					bind:permissionIds={$data.permission_ids}
					availablePermissions={permissions}
					{setFields}
					error={$errors.hasPermission?.[0]}
					data-testid="create-role-permissions"
				/>
			</div>
		</Slideover>
	</form>
{/if}

{#if editingRole}
	<form use:editForm class="space-y-6 p-1 flex flex-col h-full" data-testid="edit-role-form">
		<Slideover
			title="ロールを編集"
			primaryButtonText="更新"
			primaryButtonTypeSubmit={true}
			onClose={handleEditClose}
			loading={$editIsSubmitting}
			data-testid="edit-role-slideover"
		>
			<div class="flex-grow space-y-6">
				<Input
					id="edit-name"
					name="name"
					type="text"
					label="ロール名"
					bind:value={$editData.name}
					error={$editErrors.name?.[0]}
					placeholder="例: カスタムロール"
					variant="underlined"
				/>
				<Input
					id="edit-description"
					name="description"
					type="text"
					label="説明"
					bind:value={$editData.description}
					error={$editErrors.description?.[0]}
					placeholder="ロールの説明（任意）"
					variant="underlined"
				/>
				<PermissionSelector
					bind:permissionIds={$editData.permission_ids}
					availablePermissions={permissions}
					setFields={editSetFields}
					error={$editErrors.hasPermission?.[0]}
					data-testid="edit-role-permissions"
				/>
			</div>
		</Slideover>
	</form>
{/if}

{#if roleToDelete}
	<form use:deleteForm class="space-y-6 p-1 flex flex-col h-full" data-testid="delete-role-form">
		<Slideover
			title="ロールを削除"
			primaryButtonText="削除"
			primaryButtonTypeSubmit={true}
			onClose={handleDeleteModalClose}
			loading={$isDeleting}
			data-testid="delete-role-slideover"
		>
			<div class="flex-grow space-y-6">
				<Alert type="danger" title="この操作は元に戻せません。" data-testid="delete-role-warning">
					<p>
						削除を確認するには、ロール名<strong>「{roleToDelete.name}」</strong>を入力してください。
					</p>
					<p class="mt-2 text-sm">
						このロールが他のユーザーに割り当てられている場合、削除できません。
					</p>
				</Alert>
				<Input
					id="confirmName"
					name="confirmName"
					type="text"
					label="ロール名を入力して確認"
					bind:value={$deleteData.confirmName}
					error={$deleteErrors.confirmName?.[0]}
					required
					placeholder={roleToDelete.name}
					variant="underlined"
				/>
			</div>
		</Slideover>
	</form>
{/if}

<SettingsTabs {me} />

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
										{role.description}
									</td>
									<td
										class="px-3 py-4 text-sm text-gray-500"
										data-testid={`role-permissions-${role.id}`}
									>
										{#if role.permissions.length === 0}
											<span class="text-gray-400">権限なし</span>
										{:else}
											<div class="flex flex-wrap gap-1 items-center">
												{#each role.permissions.slice(0, 3) as permission (permission.id)}
													<Badge color="blue" size="sm" rounded="md">
														{permission.description}
													</Badge>
												{/each}
												{#if role.permissions.length > 3}
													<Tooltip class="ml-2">
														{#snippet content()}
															<div class="space-y-1">
																{#each role.permissions.slice(3) as permission (permission.id)}
																	<div class="text-xs">{permission.description}</div>
																{/each}
															</div>
														{/snippet}

														<span
															class="text-gray-400 cursor-help border-b border-dotted border-gray-300 flex items-center"
															data-testid={`role-permissions-overflow-${role.id}`}
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
											<Badge
												color="gray"
												size="sm"
												rounded="md"
												data-testid={`role-type-badge-default-${role.id}`}
											>
												デフォルト
											</Badge>
										{:else}
											<Badge
												color="green"
												size="sm"
												rounded="md"
												data-testid={`role-type-badge-custom-${role.id}`}
											>
												カスタム
											</Badge>
										{/if}
									</td>
									<td
										class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
										data-testid={`role-actions-${role.id}`}
									>
										{#if hasPermission(me, "roles.edit") && !role.is_default}
											<Button
												variant="link"
												class="mr-4 text-sm text-red-600 hover:text-red-900"
												onclick={() => handleDeleteRole(role)}
												data-testid={`delete-role-button-${role.id}`}
											>
												削除
											</Button>
											<Button
												variant="link"
												class="text-indigo-600 hover:text-indigo-900"
												onclick={() => handleEditModalOpen(role)}
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
