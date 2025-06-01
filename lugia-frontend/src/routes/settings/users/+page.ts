import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lib/fetch";

export type User = {
	id: string;
	email: string;
	name: string;
	role: "admin" | "editor";
	status: string;
	created_at: string;
	updated_at: string;
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
	const limit = parseInt(searchParams.get("limit") || "2", 10);
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

	return {
		usersPromise,
		currentPage: page,
		currentLimit: limit,
		currentSearch: search
	};
};
