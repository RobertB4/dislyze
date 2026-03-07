// Feature doc: docs/features/user-management.md
import type { PageLoad } from "./$types";
import { createLoadClient } from "$giratina/lib/api";

export function load({ fetch, params }: Parameters<PageLoad>[0]) {
	const { tenantId } = params;
	const api = createLoadClient(fetch);

	const usersPromise = api
		.GET("/tenants/{tenantID}/users", {
			params: { path: { tenantID: tenantId } }
		})
		.then(({ data }) => data!.users ?? []);

	return {
		usersPromise,
		tenantId
	};
}
