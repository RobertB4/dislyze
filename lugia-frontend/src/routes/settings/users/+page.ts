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

export const load: PageLoad = ({ fetch }) => {
	const usersPromise: Promise<User[]> = loadFunctionFetch(fetch, `/api/users`).then((res) =>
		res.json()
	);

	return { usersPromise };
};
