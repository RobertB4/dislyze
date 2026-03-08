<!-- Feature doc: docs/features/audit-logging.md -->
<script lang="ts">
	import Badge from "@dislyze/zoroark/Badge";
	import Button from "@dislyze/zoroark/Button";
	import Select from "@dislyze/zoroark/Select";
	import Layout from "$lugia/components/Layout.svelte";
	import SettingsTabs from "$lugia/routes/settings/SettingsTabs.svelte";
	import type { PageData } from "./$types";
	import { handleLoadError } from "$lugia/lib/fetch";
	import { goto } from "$app/navigation";
	import { SvelteURLSearchParams } from "svelte/reactivity";
	import { resolve } from "$app/paths";

	let { data: pageData }: { data: PageData } = $props();

	let filterResourceType = $state(pageData.filters.resourceType);
	let filterOutcome = $state(pageData.filters.outcome);
	let filterFromDate = $state(pageData.filters.fromDate);
	let filterToDate = $state(pageData.filters.toDate);

	function updateURL(page: number = pageData.currentPage) {
		const params = new SvelteURLSearchParams();
		params.set("page", page.toString());
		params.set("limit", pageData.currentLimit.toString());
		if (filterResourceType) params.set("resource_type", filterResourceType);
		if (filterOutcome) params.set("outcome", filterOutcome);
		if (filterFromDate) params.set("from_date", filterFromDate);
		if (filterToDate) params.set("to_date", filterToDate);
		goto(resolve(`/settings/audit-logs?${params.toString()}` as any), {
			replaceState: false,
			invalidate: [
				(url: URL) => {
					return url.pathname === "/api/audit-logs";
				}
			]
		});
	}

	function applyFilters() {
		updateURL(1);
	}

	function clearFilters() {
		filterResourceType = "";
		filterOutcome = "";
		filterFromDate = "";
		filterToDate = "";
		updateURL(1);
	}

	function goToPage(page: number) {
		updateURL(page);
	}

	function goToFirstPage() {
		updateURL(1);
	}

	function goToLastPage(totalPages: number) {
		updateURL(totalPages);
	}

	const outcomeMap: Record<string, { label: string; color: "green" | "red" }> = {
		success: { label: "成功", color: "green" },
		failure: { label: "失敗", color: "red" }
	};

	const resourceTypeLabels: Record<string, string> = {
		auth: "認証",
		access: "アクセス",
		user: "ユーザー",
		role: "ロール",
		ip_whitelist: "IP制限",
		tenant: "テナント"
	};

	const actionLabels: Record<string, string> = {
		login: "ログイン",
		logout: "ログアウト",
		password_changed: "パスワード変更",
		password_reset_requested: "パスワードリセット要求",
		password_reset_completed: "パスワードリセット完了",
		permission_denied: "権限拒否",
		feature_gate_blocked: "機能制限",
		ip_blocked: "IPブロック",
		invited: "招待",
		deleted: "削除",
		email_changed: "メール変更",
		roles_updated: "ロール変更",
		invite_resent: "招待再送",
		list_viewed: "一覧閲覧",
		created: "作成",
		updated: "更新",
		activated: "有効化",
		deactivated: "無効化",
		ip_added: "IP追加",
		ip_removed: "IP削除",
		ip_updated: "IP更新",
		emergency_deactivated: "緊急無効化",
		name_changed: "名前変更",
		enterprise_feature_toggled: "機能切替"
	};

	function formatDateTime(isoString: string): string {
		const date = new Date(isoString);
		return date.toLocaleString("ja-JP", {
			year: "numeric",
			month: "2-digit",
			day: "2-digit",
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit"
		});
	}

	function formatAction(resourceType: string, action: string): string {
		const resourceLabel = resourceTypeLabels[resourceType] || resourceType;
		const actionLabel = actionLabels[action] || action;
		return `${resourceLabel}: ${actionLabel}`;
	}

	function downloadCSV(auditLogs: any[]) {
		const headers = ["日時", "操作者", "メールアドレス", "操作", "結果", "IPアドレス", "詳細"];

		const rows = auditLogs.map((log) => [
			formatDateTime(log.created_at),
			sanitizeCSVValue(log.actor_name),
			sanitizeCSVValue(log.actor_email),
			formatAction(log.resource_type, log.action),
			outcomeMap[log.outcome]?.label || log.outcome,
			log.ip_address || "",
			sanitizeCSVValue(JSON.stringify(log.metadata))
		]);

		const bom = "\uFEFF";
		const csvContent =
			bom +
			[headers.join(","), ...rows.map((row) => row.map((cell) => `"${cell}"`).join(","))].join(
				"\n"
			);

		const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
		const url = URL.createObjectURL(blob);
		const link = document.createElement("a");
		link.href = url;
		link.download = `audit-logs-${new Date().toISOString().slice(0, 10)}.csv`;
		link.click();
		URL.revokeObjectURL(url);
	}

	function sanitizeCSVValue(value: string): string {
		const escaped = value.replace(/"/g, '""');
		if (/^[=+\-@\t\r]/.test(escaped)) {
			return "'" + escaped;
		}
		return escaped;
	}
</script>

<Layout me={pageData.me} pageTitle="監査ログ">
	{#await pageData.auditLogsPromise}
		<SettingsTabs me={pageData.me} />
		<div class="animate-pulse space-y-4" data-testid="audit-logs-skeleton">
			{#each Array(5)}
				<div class="h-12 bg-gray-200 rounded"></div>
			{/each}
		</div>
	{:then { audit_logs, pagination }}
		<SettingsTabs me={pageData.me} />

		<!-- Filters -->
		<div class="mb-6 space-y-4" data-testid="audit-logs-filters">
			<div class="grid grid-cols-1 md:grid-cols-4 gap-4">
				<Select
					id="resource-type-filter"
					name="resource_type"
					label="リソース種別"
					bind:value={filterResourceType}
					options={[
						{ value: "", label: "すべて" },
						...Object.entries(resourceTypeLabels).map(([v, l]) => ({ value: v, label: l }))
					]}
				/>

				<Select
					id="outcome-filter"
					name="outcome"
					label="結果"
					bind:value={filterOutcome}
					options={[
						{ value: "", label: "すべて" },
						{ value: "success", label: "成功" },
						{ value: "failure", label: "失敗" }
					]}
				/>

				<div>
					<label for="from-date" class="block text-sm font-medium text-gray-700 mb-1">開始日</label>
					<input
						id="from-date"
						type="date"
						bind:value={filterFromDate}
						class="block w-full rounded-md border-gray-300 shadow-sm focus:border-orange-500 focus:ring-orange-500 sm:text-sm"
					/>
				</div>

				<div>
					<label for="to-date" class="block text-sm font-medium text-gray-700 mb-1">終了日</label>
					<input
						id="to-date"
						type="date"
						bind:value={filterToDate}
						class="block w-full rounded-md border-gray-300 shadow-sm focus:border-orange-500 focus:ring-orange-500 sm:text-sm"
					/>
				</div>
			</div>

			<div class="flex items-center space-x-3">
				<Button
					variant="primary"
					onclick={applyFilters}
					class="text-sm"
					data-testid="apply-filters-button"
				>
					フィルター適用
				</Button>
				<Button
					variant="secondary"
					onclick={clearFilters}
					class="text-sm"
					data-testid="clear-filters-button"
				>
					クリア
				</Button>
				<Button
					variant="secondary"
					onclick={() => downloadCSV(audit_logs)}
					class="text-sm"
					data-testid="export-csv-button"
				>
					CSV出力
				</Button>
			</div>
		</div>

		<!-- Table -->
		<div class="mt-4 flow-root">
			{#if audit_logs.length === 0}
				<div class="text-center py-12" data-testid="no-audit-logs-message">
					<div class="text-gray-500 text-lg">監査ログが見つかりませんでした</div>
				</div>
			{:else}
				<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
					<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
						<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
							<table class="min-w-full divide-y divide-gray-300" data-testid="audit-logs-table">
								<thead class="bg-gray-50">
									<tr>
										<th
											scope="col"
											class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6"
											>日時</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">操作者</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">操作</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">結果</th
										>
										<th
											scope="col"
											class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
											>IPアドレス</th
										>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-200 bg-white">
									{#each audit_logs as log (log.id)}
										<tr data-testid={`audit-log-row-${log.id}`}>
											<td class="whitespace-nowrap py-4 pl-4 pr-3 text-sm text-gray-500 sm:pl-6">
												{formatDateTime(log.created_at)}
											</td>
											<td class="px-3 py-4 text-sm text-gray-900">
												<div>{log.actor_name}</div>
												<div class="text-gray-500 text-xs">
													{log.actor_email}
												</div>
											</td>
											<td class="px-3 py-4 text-sm text-gray-900">
												{formatAction(log.resource_type, log.action)}
											</td>
											<td class="whitespace-nowrap px-3 py-4 text-sm">
												<Badge color={outcomeMap[log.outcome]?.color || "green"}>
													{outcomeMap[log.outcome]?.label || log.outcome}
												</Badge>
											</td>
											<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
												{log.ip_address || "-"}
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

		<!-- Pagination -->
		{#if pagination.total > 0}
			<div class="mt-6 flex items-center justify-between" data-testid="pagination-controls">
				<div class="text-sm text-gray-700" data-testid="pagination-info">
					{pagination.total}件中 {Math.min(
						(pagination.page - 1) * pagination.limit + 1,
						pagination.total
					)} - {Math.min(pagination.page * pagination.limit, pagination.total)}件を表示
				</div>
				<div class="flex items-center space-x-2" data-testid="pagination-buttons">
					<Button
						variant="secondary"
						onclick={goToFirstPage}
						disabled={!pagination.has_prev}
						class="px-3 py-1.5 text-sm"
						data-testid="pagination-first"
					>
						«
					</Button>
					<Button
						variant="secondary"
						onclick={() => goToPage(pagination.page - 1)}
						disabled={!pagination.has_prev}
						class="px-3 py-1.5 text-sm"
						data-testid="pagination-prev"
					>
						‹
					</Button>
					<span class="px-3 py-1.5 text-sm text-gray-600" data-testid="pagination-current">
						{pagination.page} / {pagination.total_pages}
					</span>
					<Button
						variant="secondary"
						onclick={() => goToPage(pagination.page + 1)}
						disabled={!pagination.has_next}
						class="px-3 py-1.5 text-sm"
						data-testid="pagination-next"
					>
						›
					</Button>
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
	{:catch e}
		{handleLoadError(e)}
	{/await}
</Layout>
