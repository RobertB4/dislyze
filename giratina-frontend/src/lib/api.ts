import createClient, { type Middleware } from "openapi-fetch";
import type { paths } from "$giratina/schema";
import { redirect, error as svelteKitError } from "@sveltejs/kit";
import { toast } from "@dislyze/zoroark/toast";
import { KnownError } from "@dislyze/zoroark/errors";

/**
 * Creates a typed API client for use in SvelteKit load functions.
 * Must receive SvelteKit's load-function fetch for invalidate() tracking.
 * Error handling: throws SvelteKit errors for 4xx/5xx, redirects to login on 401.
 *
 * Usage:
 *   const api = createLoadClient(fetch);
 *   const data = api.GET("/tenants").then(({ data }) => data!);
 *
 * Why `data!` is safe: middleware throws on ALL error statuses before returning.
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

/**
 * Creates a typed API client for use in Svelte components for mutations.
 * Error handling: shows toast on error, redirects to login on 401.
 * Does NOT throw — callers check `!error` to decide next steps.
 *
 * Usage:
 *   const api = createMutationClient();
 *   const { error } = await api.POST("/tenants/{id}/update", {
 *     params: { path: { id: "..." } },
 *     body: { name: "...", enterprise_features: ... }
 *   });
 */
export function createMutationClient() {
	const client = createClient<paths>({
		baseUrl: "/api",
		credentials: "include"
	});

	const errorHandler: Middleware = {
		async onResponse({ response }) {
			if (response.status === 401) {
				try {
					const logoutResponse = await fetch(`/api/auth/logout`, {
						method: "POST",
						credentials: "include"
					});
					if (!logoutResponse.ok) {
						console.error(
							`api: Logout attempt failed with status ${logoutResponse.status}. Body: ${await logoutResponse.text()}`
						);
					}
				} catch (logoutAttemptError) {
					toast.showError();
					throw new Error(`api: Logout attempt network error: ${logoutAttemptError as string}`, {
						cause: logoutAttemptError
					});
				}
				window.location.href = "/auth/login";
				return;
			}

			if (response.status >= 400) {
				if (response.headers.get("content-type")?.includes("json")) {
					try {
						const cloned = response.clone();
						const body = (await cloned.json()) as { error?: string };
						if (body && typeof body.error === "string" && body.error !== "") {
							toast.showError(new KnownError(body.error));
						} else {
							toast.showError();
						}
					} catch {
						toast.showError();
					}
				} else {
					toast.showError();
				}
			}
		},

		onError({ error }) {
			console.error(`api: Network error:`, error);
			toast.showError();
			throw new Error(`api: Network error: ${String(error)}`, { cause: error });
		}
	};

	client.use(errorHandler);
	return client;
}
