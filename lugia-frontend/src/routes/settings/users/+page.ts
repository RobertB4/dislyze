// Feature doc: docs/features/user-management.md
import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lugia/lib/fetch";
import { createLoadClient } from "$lugia/lib/api";

// Not yet in OpenAPI spec — will be replaced when roles endpoints are migrated
export type Permission = {
	id: string;
	resource: string;
	action: string;
	description: string;
};

export type RoleInfo = {
	id: string;
	name: string;
	description: string;
	is_default: boolean;
	permissions: Permission[];
};

export type GetRolesResponse = {
	roles: RoleInfo[];
};

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

	const rolesPromise: Promise<GetRolesResponse> = loadFunctionFetch(fetch, `/api/users/roles`).then(
		(res) => res.json()
	);

	return {
		usersPromise,
		rolesPromise,
		currentPage: page,
		currentLimit: limit,
		currentSearch: search
	};
}
