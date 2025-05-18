import { redirect } from "@sveltejs/kit";
import type { Handle } from "@sveltejs/kit";

export const handle: Handle = async ({ event, resolve }) => {
	console.log("handle");
	const accessToken = event.cookies.get("access_token");
	const refreshToken = event.cookies.get("refresh_token");

	const pathname = event.url.pathname;
	const loggedIn = !!accessToken || !!refreshToken;

	if (!loggedIn && !pathname.startsWith("/auth/")) {
		if (pathname !== "/auth/login" && pathname !== "/auth/signup") {
			// Avoid redirect loops for login/signup
			throw redirect(303, "/auth/login");
		}
	}

	if (loggedIn && pathname.startsWith("/auth/")) {
		throw redirect(303, "/");
	}

	return resolve(event);
};
