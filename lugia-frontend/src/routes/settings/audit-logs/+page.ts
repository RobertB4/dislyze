// Feature doc: docs/features/audit-logging.md
import type { PageLoad } from "./$types";
import { createLoadClient } from "$lugia/lib/api";

export function load({ fetch, url }: Parameters<PageLoad>[0]) {
	const searchParams = url.searchParams;
	const page = parseInt(searchParams.get("page") || "1", 10);
	const limit = parseInt(searchParams.get("limit") || "50", 10);
	const actorId = searchParams.get("actor_id") || "";
	const resourceType = searchParams.get("resource_type") || "";
	const action = searchParams.get("action") || "";
	const outcome = searchParams.get("outcome") || "";
	const fromDate = searchParams.get("from_date") || "";
	const toDate = searchParams.get("to_date") || "";

	const api = createLoadClient(fetch);

	const auditLogsPromise = api
		.GET("/audit-logs", {
			params: {
				query: {
					page,
					limit,
					actor_id: actorId || undefined,
					resource_type: resourceType || undefined,
					action: action || undefined,
					outcome: outcome || undefined,
					from_date: fromDate || undefined,
					to_date: toDate || undefined
				}
			}
		})
		.then(({ data }) => data!);

	return {
		auditLogsPromise,
		currentPage: page,
		currentLimit: limit,
		filters: { actorId, resourceType, action, outcome, fromDate, toDate }
	};
}
