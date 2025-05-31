import { redirect, error as svelteKitError } from "@sveltejs/kit";
import { KnownError } from "./errors";
import { toast } from "$components/Toast/toast";

/**
 * For use in the `load` function to fetch data for the page.
 * Do not catch the error of this function.
 * Catching the error prevents SvelteKit from handling it.
 * If you need to catch the error, make sure to rethrow it.
 */
export async function loadFunctionFetch(
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

/**
 * For use in Svelte components for mutations (POST, PUT, DELETE).
 * Handles common error cases and 401 redirection.
 * It is not needed to catch the error of this function unless there is a reason to.
 */
export async function mutationFetch(
	url: string,
	options?: RequestInit
): Promise<{ response: Response; success: boolean }> {
	let response: Response;
	let success = false;
	const requestOptions = options ?? {};
	requestOptions.credentials = requestOptions.credentials ?? "include";

	try {
		response = await fetch(url, requestOptions);
	} catch (networkError) {
		toast.showError();
		throw new Error(
			`mutationFetch: Network error for URL ${url.toString()}: ${networkError as string}`
		);
	}

	if (response.status === 401) {
		try {
			const logoutResponse = await fetch(`/api/auth/logout`, {
				method: "POST",
				credentials: "include"
			});
			if (!logoutResponse.ok) {
				console.error(
					`mutationFetch: Logout attempt failed with status ${logoutResponse.status}. Body: ${await logoutResponse.text()}`
				);
			}
		} catch (logoutAttemptError) {
			toast.showError();
			throw new Error(
				`mutationFetch: Logout attempt network error: ${logoutAttemptError as string}`
			);
		}

		window.location.href = "/auth/login";
	}

	if (
		response.status >= 400 &&
		response.headers.get("content-type")?.includes("application/json")
	) {
		try {
			const clonedResponse = response.clone();
			const body = (await clonedResponse.json()) as { error?: string };
			if (body && typeof body.error === "string") {
				toast.showError(new KnownError(body.error));
				success = false;
			} else {
				toast.showError();
				success = false;
			}
		} catch (jsonError) {
			console.warn(
				"mutationFetch: Could not parse JSON body for error key or body not JSON:",
				jsonError
			);
		}
	}

	if (
		response.status >= 400 &&
		!response.headers.get("content-type")?.includes("application/json")
	) {
		toast.showError();
		success = false;
	}

	if (response.status >= 200 && response.status <= 204) {
		success = true;
	}

	return { response, success };
}
