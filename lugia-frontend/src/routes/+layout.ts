import type { LayoutLoad } from "./$types";
import { redirect, error as svelteKitError } from "@sveltejs/kit";
import { forceUpdateMeCache, meCache, type Me } from "@dislyze/zoroark/meCache";
import { get } from "svelte/store";

export const ssr = false;
export const prerender = false;

/**
 * For use in the `load` function to fetch data for the page.
 * Do not catch the error of this function.
 * Catching the error prevents SvelteKit from handling it.
 * If you need to catch the error, make sure to rethrow it.
 */
async function loadFunctionFetch(
	loadEventFetch: typeof fetch,
	url: string,
	options?: RequestInit
): Promise<Response> {
	let response: Response;
	try {
		const requestOptions = options ?? {};
		requestOptions.credentials = requestOptions.credentials ?? "include";
		response = await loadEventFetch(url, requestOptions);
	} catch (networkError) {
		console.error(`loadFunctionFetch: Network error for URL ${url.toString()}:`, networkError);
		throw svelteKitError(
			503,
			"ネットワーク接続に問題があるか、サーバーが応答しませんでした。接続を確認し、再度お試しください。"
		);
	}

	if (response.status >= 500) {
		console.error(
			`loadFunctionFetch: Server error for URL ${response.url}, status ${response.status}`
		);
		throw svelteKitError(
			response.status,
			"サーバーでエラーが発生しました。時間をおいて再度お試しください。"
		);
	}

	if (response.status === 404) {
		console.error(`loadFunctionFetch: Not found for URL ${response.url}`);
		throw svelteKitError(404, "ページが見つかりません。");
	}

	if (response.status === 403) {
		console.error(`loadFunctionFetch: Forbidden for URL ${response.url}`);
		throw svelteKitError(403, "権限がありません。");
	}

	if (response.status === 401) {
		try {
			const logoutResponse = await loadEventFetch(`/api/auth/logout`, {
				method: "POST",
				credentials: "include"
			});
			if (!logoutResponse.ok) {
				console.error(
					`loadFunctionFetch: Logout attempt failed with status ${logoutResponse.status} after 401. Body: ${await logoutResponse.text()}`
				);
				throw svelteKitError(
					logoutResponse.status,
					"サーバーでエラーが発生しました。時間をおいて再度お試しください。"
				);
			}
		} catch (logoutAttemptError) {
			console.error(
				`loadFunctionFetch: Network error or other issue during logout attempt for URL ${url.toString()}:`,
				logoutAttemptError
			);
			throw svelteKitError(
				503,
				"ネットワーク接続に問題があるか、サーバーが応答しませんでした。接続を確認し、再度お試しください。"
			);
		}

		throw redirect(307, "/auth/login");
	}

	return response;
}

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

export async function load({ fetch, url }: Parameters<LayoutLoad>[0]) {
	// This page only gets opened when the user is locked out due to their ip not being in the whitelist.
	// If we call /api/me down below, it will return 403 because of this.
	// Therefore, we need to skip the /api/me call for this page.
	if (url.pathname.startsWith("/settings/ip-whitelist/emergency-deactivate")) {
		return { me: null as any };
	}

	// user needs to be able to display /error when they are locked out
	if (url.pathname === "/error") {
		return { me: null as any };
	}

	if (!get(forceUpdateMeCache)) {
		if (typeof window !== "undefined") {
			if (get(meCache)) {
				return { me: get(meCache) };
			}
		}
	}

	if (get(forceUpdateMeCache)) {
		forceUpdateMeCache.set(false);
	}

	let me!: Me;

	if (url.pathname.startsWith("/auth")) {
		// SCENARIO A: User is on an /auth page (e.g., /auth/login, /auth/signup)
		try {
			const response = await loadFunctionFetch(fetch, `/api/me`);
			if (response.ok) {
				const user = (await response.json()) as Me;
				// Check for a valid user identifier (user_id from your meS.ts Me interface)
				if (user && user.user_id) {
					throw redirect(307, "/"); // User is logged in, redirect from /auth page
				}
				// Got 200 OK but no valid user data from /me (e.g. API returns {} or specific non-error for no session)
				me = null as any;
			} else {
				// Response was not .ok (e.g. 400, 422 from /users/me) and not an error loadFunctionFetch throws for.
				// For an /auth path, this is unexpected if /me is supposed to 401 for unauthenticated.
				// Treat as not logged in for the purpose of an auth page.
				console.warn(
					`+layout.ts: On auth path, /users/me returned !ok (${response.status}). Assuming not logged in. Allowing auth page render.`
				);
				me = null as any;
			}
		} catch (err: unknown) {
			// If loadFunctionFetch threw a redirect to /auth/login (e.g. from a 401 on /users/me)
			if (isRedirect(err) && err.location === "/auth/login") {
				me = null as any; // User not logged in, allow auth page
			} else {
				// For any other error (e.g., 500 server error on /users/me, network error from loadFunctionFetch)
				console.error("+layout.ts: On auth path, unexpected error during /users/me fetch:", err);
				throw err; // Re-throw to SvelteKit (will render +error.svelte or handle other redirects)
			}
		}
	} else if (url.pathname.startsWith("/verify")) {
		// SCENARIO B: User is on a /verify page (e.g., /verify/change-email)
		// Never redirect automatically - always allow the verify page to render
		// The verify page itself will handle showing appropriate messages and redirects based on auth status
		try {
			const response = await loadFunctionFetch(fetch, `/api/me`);
			if (response.ok) {
				me = await response.json();
			} else {
				// User not authenticated, but still allow verify page to render
				me = null as any;
			}
		} catch (err: unknown) {
			// If loadFunctionFetch threw a redirect to /auth/login (e.g. from a 401 on /users/me)
			if (isRedirect(err) && err.location === "/auth/login") {
				me = null as any; // User not logged in, allow verify page to render
			} else {
				// For any other error (e.g., 500 server error on /users/me, network error from loadFunctionFetch)
				console.error("+layout.ts: On verify path, unexpected error during /users/me fetch:", err);
				throw err; // Re-throw to SvelteKit (will render +error.svelte or handle other redirects)
			}
		}
	} else {
		// SCENARIO C: User is on a protected page (NOT /auth or /verify)
		try {
			const response = await loadFunctionFetch(fetch, `/api/me`);

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

			me = await response.json();
		} catch (err: unknown) {
			let status = 500;
			let message = "処理中に予期せぬエラーが発生しました。";

			const errorObj = err as {
				status?: number;
				message?: string;
				body?: { message?: string };
				location?: string;
			};

			// Handle redirects (like 401 to /auth/login
			if (errorObj.location) {
				throw redirect(307, errorObj.location);
			}

			if (errorObj.status) {
				status = errorObj.status;
			}

			if (errorObj.message) {
				message = errorObj.message;
			}

			if (errorObj.body?.message) {
				message = errorObj.body.message;
			}

			if (err instanceof Error) {
				message = err.message;
			}

			throw redirect(307, `/error?status=${status}&message=${encodeURIComponent(message)}`);
		}
	}
	return { me };
}
