import type { LayoutLoad } from "./$types";
import { PUBLIC_API_URL } from "$env/static/public";
import { loadFunctionFetch } from "$lib/fetch";
import { redirect, error as svelteKitError } from "@sveltejs/kit";
import type { User } from "$lib/stores/meStore";

// Helper type guard to check if an error is a SvelteKit Redirect
function isRedirect(error: unknown): error is import("@sveltejs/kit").Redirect {
	if (typeof error !== "object" || error === null) return false;
	const errObj = error as Record<string, unknown>;
	return (
		typeof errObj.status === "number" &&
		errObj.status >= 300 &&
		errObj.status < 400 &&
		typeof errObj.location === "string"
	);
}

export const ssr = false;
export const prerender = false;

export const load: LayoutLoad = async ({ fetch, url }) => {
	let initialUser: User | null = null;

	if (url.pathname.startsWith("/auth")) {
		// SCENARIO A: User is on an /auth page (e.g., /auth/login)
		try {
			const response = await loadFunctionFetch(fetch, `${PUBLIC_API_URL}/me`);
			if (response.ok) {
				const user = (await response.json()) as User;
				// Check for a valid user identifier (user_id from your meStore.ts User interface)
				if (user && user.user_id) {
					throw redirect(307, "/"); // User is logged in, redirect from /auth page
				}
				// Got 200 OK but no valid user data from /me (e.g. API returns {} or specific non-error for no session)
				initialUser = null;
			} else {
				// Response was not .ok (e.g. 400, 422 from /users/me) and not an error loadFunctionFetch throws for.
				// For an /auth path, this is unexpected if /me is supposed to 401 for unauthenticated.
				// Treat as not logged in for the purpose of an auth page.
				console.warn(
					`+layout.ts: On auth path, /users/me returned !ok (${response.status}). Assuming not logged in. Allowing auth page render.`
				);
				initialUser = null;
			}
		} catch (err: unknown) {
			// If loadFunctionFetch threw a redirect to /auth/login (e.g. from a 401 on /users/me)
			if (isRedirect(err) && err.location === "/auth/login") {
				initialUser = null; // User not logged in, allow auth page
			} else {
				// For any other error (e.g., 500 server error on /users/me, network error from loadFunctionFetch)
				console.error("+layout.ts: On auth path, unexpected error during /users/me fetch:", err);
				throw err; // Re-throw to SvelteKit (will render +error.svelte or handle other redirects)
			}
		}
	} else {
		// SCENARIO B: User is on a protected page (NOT /auth)
		try {
			const response = await loadFunctionFetch(fetch, `${PUBLIC_API_URL}/me`);

			if (!response.ok) {
				// loadFunctionFetch throws for 401,403,404,5xx.
				// This handles other !response.ok cases (e.g., 400, 422 from /users/me).
				const errorBody = await response.text();
				console.error(
					`+layout.ts: Protected path, /users/me fetch !ok (status ${response.status}). Body: ${errorBody}`
				);
				throw svelteKitError(
					response.status,
					`ユーザーの読み込みに失敗しました: ${response.statusText || response.status}`
				);
			}

			initialUser = (await response.json()) as User;
		} catch (err: unknown) {
			// This re-throws redirects (like 401 to /auth/login from loadFunctionFetch) & SvelteKit errors.
			console.error("+layout.ts: Protected path, error/redirect during /users/me fetch:", err);
			throw err;
		}
	}
	return { initialUser };
};
