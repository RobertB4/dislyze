import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lib/fetch";

export type RoleInfo = {
	id: string;
	name: string;
	description: string;
	is_default: boolean;
	permissions: string[];
};

export type PermissionInfo = {
	id: string;
	resource: string;
	action: string;
	description: string;
};

export type GetRolesResponse = {
	roles: RoleInfo[];
};

export type GetPermissionsResponse = {
	permissions: PermissionInfo[];
};

export const load: PageLoad = ({ fetch }) => {
	const rolesPromise: Promise<GetRolesResponse> = loadFunctionFetch(fetch, `/api/roles`).then(
		(res) => res.json()
	);

	const permissionsPromise: Promise<GetPermissionsResponse> = loadFunctionFetch(
		fetch,
		`/api/roles/permissions`
	).then((res) => res.json());

	return {
		rolesPromise,
		permissionsPromise
	};
};
