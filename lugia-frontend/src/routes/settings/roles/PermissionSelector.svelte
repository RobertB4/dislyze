<script lang="ts">
	import Pill from "$components/Pill.svelte";
	import type { PermissionInfo } from "./+page";

	let {
		permissionIds = $bindable(),
		availablePermissions,
		setFields,
		error,
		"data-testid": dataTestid
	}: {
		permissionIds: string[];
		availablePermissions: PermissionInfo[];
		setFields: (
			key: "permission_ids" | "name" | "description" | "hasPermission" | `permission_ids.${number}`,
			value: any
		) => void;
		error?: string;
		"data-testid"?: string;
	} = $props();

	function getResourceDisplayName(resource: string): string {
		const resourceLabels: Record<string, string> = {
			users: "ユーザー管理",
			roles: "ロール管理",
			tenant: "テナント設定"
		};
		return resourceLabels[resource] || resource;
	}

	function getCurrentSelection(resource: string): "none" | "view" | "edit" {
		// Find which permission from this resource is currently selected
		const resourcePermissions = availablePermissions.filter((p) => p.resource === resource);

		for (const permission of resourcePermissions) {
			if (permissionIds.includes(permission.id)) {
				return permission.action as "view" | "edit";
			}
		}

		return "none";
	}

	function selectOption(resource: string, option: "none" | "view" | "edit") {
		const currentIds = [...permissionIds];

		// Remove all existing permissions for this resource
		const resourcePermissions = availablePermissions.filter((p) => p.resource === resource);
		resourcePermissions.forEach((permission) => {
			const index = currentIds.indexOf(permission.id);
			if (index > -1) {
				currentIds.splice(index, 1);
			}
		});

		// Add the selected permission if not "none"
		if (option !== "none") {
			const selectedPermission = resourcePermissions.find((p) => p.action === option);
			if (selectedPermission) {
				currentIds.push(selectedPermission.id);
			}
		}

		setFields("permission_ids", currentIds);
	}

	function getActionLabel(action: "none" | "view" | "edit"): string {
		const labels = {
			none: "なし",
			view: "閲覧",
			edit: "編集"
		};
		return labels[action];
	}

	// Group permissions by resource for better UI organization
	let groupedPermissions = $derived(() => {
		const groups = new Map<string, PermissionInfo[]>();

		availablePermissions.forEach((permission) => {
			if (!groups.has(permission.resource)) {
				groups.set(permission.resource, []);
			}
			groups.get(permission.resource)!.push(permission);
		});

		return Array.from(groups.entries()).map(([resource, permissions]) => ({
			resource,
			displayName: getResourceDisplayName(resource),
			permissions
		}));
	});
</script>

<div class="space-y-6" data-testid={dataTestid}>
	<div class="text-xs text-gray-500">
		<p><strong>編集権限:</strong> 閲覧権限も自動的に含まれます</p>
		<p>必要な権限を選択してください</p>
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
		{#each groupedPermissions() as group (group.resource)}
			<div
				class="border border-gray-200 rounded-lg p-4"
				data-testid={`permission-group-${group.resource}`}
			>
				<div class="flex items-center justify-between">
					<h4
						class="text-sm font-medium text-gray-900"
						data-testid={`permission-group-title-${group.resource}`}
					>
						{group.displayName}
					</h4>
					<div class="flex gap-2">
					<!-- None option -->
					<Pill
						selected={getCurrentSelection(group.resource) === "none"}
						onclick={() => selectOption(group.resource, "none")}
						variant="orange"
						data-testid={`permission-${group.resource}-none`}
					>
						{getActionLabel("none")}
					</Pill>

					<!-- View option (if available) -->
					{#if group.permissions.some((p) => p.action === "view")}
						<Pill
							selected={getCurrentSelection(group.resource) === "view"}
							onclick={() => selectOption(group.resource, "view")}
							variant="orange"
							data-testid={`permission-${group.resource}-view`}
						>
							{getActionLabel("view")}
						</Pill>
					{/if}

					<!-- Edit option (if available) -->
					{#if group.permissions.some((p) => p.action === "edit")}
						<Pill
							selected={getCurrentSelection(group.resource) === "edit"}
							onclick={() => selectOption(group.resource, "edit")}
							variant="orange"
							data-testid={`permission-${group.resource}-edit`}
						>
							{getActionLabel("edit")}
						</Pill>
					{/if}
					</div>
				</div>
			</div>
		{/each}
	</div>
</div>
