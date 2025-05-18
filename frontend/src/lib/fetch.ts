import { goto } from "$app/navigation";
import { PUBLIC_API_URL } from "$env/static/public";

export async function handleFetch(
	fetchFunction: typeof fetch,
	url: string | URL | Request,
	options?: RequestInit
): Promise<Response> {
	const response = await fetchFunction(url, options);
	if (response.status === 401) {
		console.log("inside");
		try {
			const res = await fetchFunction(`${PUBLIC_API_URL}/auth/logout`, {
				method: "POST",
				credentials: "include"
			});

			if (!res.ok) {
				console.log("not ok");
				// TODO: show 500 error screen
			}
		} catch (logoutError) {
			console.error("Logout request failed:", logoutError);
			// TODO: show 500 error screen
		}
		goto("/auth/login");
		throw new Error("Session expired. Redirecting to login.");
	}

	return response;
}
