import type { PageLoad } from "./$types";

export function load({ url }: Parameters<PageLoad>[0]) {
	const error = url.searchParams.get("error");
	const email = url.searchParams.get("email");

	return {
		error,
		email
	};
}
