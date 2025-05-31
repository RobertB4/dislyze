import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lib/fetch";

type User = {
	id: string;
	email: string;
	name: string;
	role: string;
	status: string;
	created_at: string;
	updated_at: string;
};

export const load: PageLoad = async ({ fetch }) => {
	const response = await loadFunctionFetch(fetch, `/api/users`);

	const users: User[] = await response.json();
	return { users };
};
