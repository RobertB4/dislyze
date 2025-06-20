import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lib/fetch";

export interface User {
	id: string;
	name: string;
	email: string;
	status: string;
}

export interface GetUsersByTenantResponse {
	users: User[];
}

export const load: PageLoad = ({ fetch, params }) => {
	const { tenantId } = params;

	const usersPromise = loadFunctionFetch(fetch, `/api/tenants/${tenantId}/users`)
		.then((response) => response.json())
		.then((data: GetUsersByTenantResponse) => data);

	return {
		usersPromise,
		tenantId
	};
};
