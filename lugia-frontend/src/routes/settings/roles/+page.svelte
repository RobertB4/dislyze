<script lang="ts">
	import { Button } from "@dislyze/zoroark";
	import Layout from "$components/Layout.svelte";
	import Skeleton from "./Skeleton.svelte";
	import RolesTable from "./RolesTable.svelte";
	import type { PageData } from "./$types";
	import { hasPermission } from "$lib/authz";

	let { data: pageData }: { data: PageData } = $props();

	let isCreateSlideoverOpen = $state(false);
</script>

<Layout
	me={pageData.me}
	pageTitle="ロール管理"
	promises={{
		rolesResponse: pageData.rolesPromise,
		permissionsResponse: pageData.permissionsPromise
	}}
>
	{#snippet buttons()}
		{#if hasPermission(pageData.me, "roles.edit")}
			<Button
				type="button"
				variant="primary"
				onclick={() => (isCreateSlideoverOpen = true)}
				data-testid="add-role-button"
			>
				ロールを追加
			</Button>
		{/if}
	{/snippet}

	{#snippet skeleton()}
		<Skeleton />
	{/snippet}

	{#snippet children({ rolesResponse, permissionsResponse })}
		{@const { roles } = rolesResponse}
		{@const { permissions } = permissionsResponse}

		<RolesTable me={pageData.me} {roles} {permissions} bind:isCreateSlideoverOpen />
	{/snippet}
</Layout>
