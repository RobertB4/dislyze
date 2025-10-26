import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
	const message = url.searchParams.get("message");

	return {
		message
	};
};
