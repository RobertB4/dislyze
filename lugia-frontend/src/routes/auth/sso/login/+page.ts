import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
	const error = url.searchParams.get("error");

	return {
		error
	};
};
