import createClient, { type Middleware } from "openapi-fetch";
import type { paths } from "$lugia/schema";
import { redirect, error as svelteKitError } from "@sveltejs/kit";

/**
 * Creates a typed API client for use in SvelteKit load functions.
 * Must receive SvelteKit's load-function fetch for invalidate() tracking.
 * Error handling mirrors loadFunctionFetch: throws SvelteKit errors/redirects.
 *
 * Usage:
 *   const api = createLoadClient(fetch);
 *   const data = api.GET("/users", { params: { query: { page: 1 } } }).then(({ data }) => data!);
 *
 * Why `data!` is safe: openapi-fetch returns { data, error, response } as a discriminated union
 * where data is T | undefined. The middleware below throws on ALL error statuses (401, 403, 404,
 * 5xx, network errors) before openapi-fetch returns. So if the call returns at all, it succeeded
 * and data is always defined. TypeScript can't infer this, so the non-null assertion is needed.
 */
export function createLoadClient(loadEventFetch: typeof fetch) {
	const client = createClient<paths>({
		baseUrl: "/api",
		fetch: loadEventFetch,
		credentials: "include"
	});

	const errorHandler: Middleware = {
		async onResponse({ response }) {
			if (response.status >= 500) {
				console.error(`api: Server error for URL ${response.url}, status ${response.status}`);
				throw svelteKitError(
					response.status,
					"サーバーでエラーが発生しました。時間をおいて再度お試しください。"
				);
			}

			if (response.status === 404) {
				console.error(`api: Not found for URL ${response.url}`);
				throw svelteKitError(404, "ページが見つかりません。");
			}

			if (response.status === 403) {
				console.error(`api: Forbidden for URL ${response.url}`);
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
							`api: Logout attempt failed with status ${logoutResponse.status} after 401. Body: ${await logoutResponse.text()}`
						);
						throw svelteKitError(
							logoutResponse.status,
							"サーバーでエラーが発生しました。時間をおいて再度お試しください。"
						);
					}
				} catch (logoutAttemptError) {
					if (
						logoutAttemptError &&
						typeof logoutAttemptError === "object" &&
						"status" in logoutAttemptError
					) {
						throw logoutAttemptError;
					}
					console.error(`api: Network error during logout attempt:`, logoutAttemptError);
					throw svelteKitError(
						503,
						"ネットワーク接続に問題があるか、サーバーが応答しませんでした。接続を確認し、再度お試しください。"
					);
				}

				throw redirect(307, "/auth/login");
			}
		},

		onError({ error }) {
			console.error(`api: Network error:`, error);
			throw svelteKitError(
				503,
				"ネットワーク接続に問題があるか、サーバーが応答しませんでした。接続を確認し、再度お試しください。"
			);
		}
	};

	client.use(errorHandler);
	return client;
}
