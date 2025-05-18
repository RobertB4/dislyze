import type { PageLoad } from "./$types";
import { PUBLIC_API_URL } from "$env/static/public";
import { handleFetch } from "$lib/fetch";

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
	const response = await handleFetch(fetch, `${PUBLIC_API_URL}/users`, {
		credentials: "include"
	});

	const users: User[] = await response.json();
	return { users };
};
