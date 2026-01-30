import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
	const error = url.searchParams.get("error");
	const email = url.searchParams.get("email");

	return {
		error,
		email
	};
};
