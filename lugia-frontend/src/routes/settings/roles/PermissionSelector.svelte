<script module lang="ts">
	export type PermissionLevel = "none" | "view" | "edit";

	export type ResourcePermissions = {
		users: PermissionLevel;
		roles: PermissionLevel;
		tenant: PermissionLevel;
	};

	export type PermissionInfo = {
		id: string;
		resource: string;
		action: string;
		description: string;
	};
</script>

<script lang="ts">
	import Pill from "$components/Pill.svelte";

	let {
		permissions = $bindable(),
		availablePermissions,
		setFields,
		error,
		"data-testid": dataTestid
	}: {
		permissions: ResourcePermissions;
		availablePermissions: PermissionInfo[];
		setFields: (
			fields: Record<
				| "permissions"
				| "name"
				| "description"
				| "hasPermission"
				| "permissions.tenant"
				| "permissions.users"
				| "permissions.roles",
				string
			>,
			value: string
		) => void;
		error?: string;
		"data-testid"?: string;
	} = $props();
	let resources = $derived(() => {
		const resourceMap = new Map<string, string>();

		const resourceLabels: Record<string, string> = {
			users: "ユーザー管理",
			roles: "ロール管理",
			tenant: "テナント設定"
		};

		availablePermissions.forEach((permission) => {
			if (!resourceMap.has(permission.resource)) {
				resourceMap.set(
					permission.resource,
					resourceLabels[permission.resource] || permission.resource
				);
			}
		});

		return Array.from(resourceMap.entries()).map(([key, label]) => ({
			key,
			label
		}));
	});

	const levels = [
		{ value: "none", label: "なし" },
		{ value: "view", label: "閲覧" },
		{ value: "edit", label: "編集" }
	] as const;
</script>

<div class="space-y-6" data-testid={dataTestid}>
	<div class="text-xs text-gray-500">
		<p><strong>なし:</strong> 該当機能にアクセスできません</p>
		<p><strong>閲覧:</strong> 一覧表示や詳細確認ができます</p>
		<p><strong>編集:</strong> 閲覧権限に加えて、作成・更新・削除ができます</p>
	</div>

	<div>
		<h3 class="text-sm font-medium text-gray-700 mb-4">権限設定</h3>
		{#if error}
			<div class="text-sm text-red-600 mb-4" data-testid="permissions-error">
				{error}
			</div>
		{/if}
	</div>

	<div class="space-y-4">
		{#each resources() as resource (resource.key)}
			<div class="border border-gray-200 rounded-lg p-4">
				<div class="flex items-center justify-between">
					<h4 class="text-sm font-medium text-gray-900 min-w-0 flex-shrink-0">
						{resource.label}
					</h4>
					<div class="flex gap-2 ml-4">
						{#each levels as level (level.value)}
							<Pill
								selected={permissions[resource.key as "tenant" | "users" | "roles"] === level.value}
								onclick={() => {
									setFields(`permissions.${resource.key}` as any, level.value);
								}}
								variant="orange"
								data-testid={`permission-${resource.key}-${level.value}`}
							>
								{level.label}
							</Pill>
						{/each}
					</div>
				</div>
			</div>
		{/each}
	</div>
</div>
