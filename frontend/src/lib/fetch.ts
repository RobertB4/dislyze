import { redirect, error as svelteKitError } from "@sveltejs/kit";
import { PUBLIC_API_URL } from "$env/static/public";

/**
 * For use in the `load` function to fetch data for the page.
 * Do not catch the error of this function.
 * Catching the error prevents SvelteKit from handling it.
 * If you need to catch the error, make sure to rethrow it.
 */
export async function loadFunctionFetch(
	loadEventFetch: typeof fetch,
	url: string | URL | Request,
	options?: RequestInit
): Promise<Response> {
	let response: Response;
	try {
		const opt = options ?? {};
		opt.credentials = "include";
		response = await loadEventFetch(url, opt);
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

	if (response.status === 403) {
		console.error(`loadFunctionFetch: Forbidden for URL ${response.url}`);
		throw svelteKitError(403, "権限がありません。");
	}

	if (response.status === 401) {
		console.log(
			`loadFunctionFetch: Received 401 from ${response.url}. Attempting logout before redirecting to login.`
		);
		try {
			const logoutResponse = await loadEventFetch(`${PUBLIC_API_URL}/auth/logout`, {
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
