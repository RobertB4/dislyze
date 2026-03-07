// Feature doc: docs/features/rbac.md
import type { PageLoad } from "./$types";
import { createLoadClient } from "$lugia/lib/api";

export function load({ fetch }: Parameters<PageLoad>[0]) {
	const api = createLoadClient(fetch);

	const rolesPromise = api.GET("/roles").then(({ data }) => data!);
	const permissionsPromise = api.GET("/roles/permissions").then(({ data }) => data!);

	return {
		rolesPromise,
		permissionsPromise
	};
}
