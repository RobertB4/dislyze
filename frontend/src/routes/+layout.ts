import type { LayoutLoad } from "./$types";
import { PUBLIC_API_URL } from "$env/static/public";

export const ssr = false;
export const prerender = false;

export const load: LayoutLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${PUBLIC_API_URL}/me`);

		if (response.status === 401 || response.status === 403) {
			return {
				currentUser: null,
				error: { status: response.status, message: "Not authenticated" }
			};
		}

		if (!response.ok) {
			console.error("Failed to fetch /me, status:", response.status);
			return {
				currentUser: null,
				error: { status: response.status, message: "Failed to load user profile" }
			};
		}
		const currentUser = await response.json();
		return { currentUser, error: null };
	} catch (e) {
		console.error("Network error or other exception fetching /me:", e);
		return {
			currentUser: null,
			error: { message: "Failed to connect to server to load user profile" }
		};
	}
};
