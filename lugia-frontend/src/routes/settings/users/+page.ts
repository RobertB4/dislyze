import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lib/fetch";

export type UserRole = {
	id: string;
	name: string;
	description: string;
};

export type User = {
	id: string;
	email: string;
	name: string;
	roles: UserRole[];
	status: string;
	created_at: string;
	updated_at: string;
};

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

export type PaginationMetadata = {
	page: number;
	limit: number;
	total: number;
	total_pages: number;
	has_next: boolean;
	has_prev: boolean;
};

export type GetUsersResponse = {
	users: User[];
	pagination: PaginationMetadata;
};

export const load: PageLoad = ({ fetch, url }) => {
	const searchParams = url.searchParams;
	const page = parseInt(searchParams.get("page") || "1", 10);
	const limit = parseInt(searchParams.get("limit") || "50", 10);
	const search = searchParams.get("search") || "";

	const queryParams = new URLSearchParams();
	queryParams.set("page", page.toString());
	queryParams.set("limit", limit.toString());
	if (search) {
		queryParams.set("search", search);
	}

	const usersPromise: Promise<GetUsersResponse> = loadFunctionFetch(
		fetch,
		`/api/users?${queryParams.toString()}`
	).then((res) => res.json());

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
};
