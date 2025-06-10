import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lib/fetch";

export type RoleInfo = {
	id: string;
	name: string;
	description: string;
	is_default: boolean;
	permissions: string[];
};

export type GetRolesResponse = {
	roles: RoleInfo[];
};

export const load: PageLoad = ({ fetch }) => {
	const rolesPromise: Promise<GetRolesResponse> = loadFunctionFetch(fetch, `/api/roles`).then(
		(res) => res.json()
	);

	return {
		rolesPromise
	};
};