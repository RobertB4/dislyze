<!-- Feature doc: docs/features/rbac.md -->
<script lang="ts">
	import Button from "@dislyze/zoroark/Button";
	import Layout from "$lugia/components/Layout.svelte";
	import Skeleton from "$lugia/routes/settings/roles/Skeleton.svelte";
	import RolesTable from "$lugia/routes/settings/roles/RolesTable.svelte";
	import type { PageData } from "./$types";
	import { hasPermission } from "$lugia/lib/authz";
	import { handleLoadError } from "$lugia/lib/fetch";

	let { data: pageData }: { data: PageData } = $props();

	let isCreateSlideoverOpen = $state(false);
</script>

<Layout me={pageData.me} pageTitle="ロール管理">
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

	{#await Promise.all([pageData.rolesPromise, pageData.permissionsPromise])}
		<Skeleton />
	{:then [{ roles }, { permissions }]}
		<RolesTable me={pageData.me} {roles} {permissions} bind:isCreateSlideoverOpen />
	{:catch e}
		{handleLoadError(e)}
	{/await}
</Layout>
