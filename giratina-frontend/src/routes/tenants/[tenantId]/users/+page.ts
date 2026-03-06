import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$giratina/lib/fetch";

export interface User {
	id: string;
	name: string;
	email: string;
	status: string;
}

export interface GetUsersByTenantResponse {
	users: User[];
}

export function load({ fetch, params }: Parameters<PageLoad>[0]) {
	const { tenantId } = params;

	const usersPromise = loadFunctionFetch(fetch, `/api/tenants/${tenantId}/users`)
		.then((response) => response.json())
		.then((data: GetUsersByTenantResponse) => data);

	return {
		usersPromise,
		tenantId
	};
}
