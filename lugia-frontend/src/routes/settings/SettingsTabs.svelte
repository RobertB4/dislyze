<script lang="ts">
	import { page } from "$app/state";
	import type { Me } from "@dislyze/zoroark";
	import { hasFeature, hasPermission } from "$lib/authz";
	import { resolve } from "$app/paths";

	let { me }: { me: Me } = $props();

	const tabs = [
		{
			name: "プロフィール",
			href: "/settings/profile",
			id: "profile"
		},
		...(hasPermission(me, "users.view")
			? [
					{
						name: "ユーザー管理",
						href: "/settings/users",
						id: "users"
					}
				]
			: []),
		...(hasPermission(me, "roles.view") && hasFeature(me, "rbac")
			? [
					{
						name: "ロール管理",
						href: "/settings/roles",
						id: "roles"
					}
				]
			: []),
		...(hasPermission(me, "ip_whitelist.view") && hasFeature(me, "ip_whitelist")
			? [
					{
						name: "IPアドレス制限",
						href: "/settings/ip-whitelist",
						id: "ip-whitelist"
					}
				]
			: [])
	];

	function isActiveTab(href: string): boolean {
		return page.route.id?.startsWith(href) ?? false;
	}
</script>

<div class="border-b border-gray-200 mb-6">
	<nav class="-mb-px flex space-x-8" aria-label="Tabs" data-testid="settings-tabs">
		{#each tabs as tab (tab.id)}
			<a
				href={resolve(tab.href)}
				class={`
					whitespace-nowrap py-2 px-1 border-b-2 text-sm transition-all duration-200
					${
						isActiveTab(tab.href)
							? "border-orange-500 text-orange-600 font-bold"
							: "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 font-medium"
					}
				`}
				aria-current={isActiveTab(tab.href) ? "page" : undefined}
				data-testid={`settings-tab-${tab.id}`}
			>
				{tab.name}
			</a>
		{/each}
	</nav>
</div>
