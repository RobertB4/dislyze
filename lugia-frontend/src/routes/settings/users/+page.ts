// Feature doc: docs/features/user-management.md
import type { PageLoad } from "./$types";
import { createLoadClient } from "$lugia/lib/api";

export function load({ fetch, url }: Parameters<PageLoad>[0]) {
	const searchParams = url.searchParams;
	const page = parseInt(searchParams.get("page") || "1", 10);
	const limit = parseInt(searchParams.get("limit") || "50", 10);
	const search = searchParams.get("search") || "";

	const api = createLoadClient(fetch);

	const usersPromise = api
		.GET("/users", {
			params: { query: { page, limit, ...(search ? { search } : {}) } }
		})
		.then(({ data }) => data!);

	const rolesPromise = api.GET("/users/roles").then(({ data }) => data!);

	return {
		usersPromise,
		rolesPromise,
		currentPage: page,
		currentLimit: limit,
		currentSearch: search
	};
}
