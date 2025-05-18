import { goto } from "$app/navigation";
import { PUBLIC_API_URL } from "$env/static/public";
import { errorStore } from "$lib/errors";
import { get } from "svelte/store";

export async function handleFetch(
	fetchFunction: typeof fetch,
	url: string | URL | Request,
	options?: RequestInit
): Promise<Response> {
	const response = await fetchFunction(url, options);

	if (response.status >= 500) {
		errorStore.setError(
			response.status,
			"サーバーでエラーが発生しました。時間をおいて再度お試しください。"
		);
		throw new Error(`Server error: ${response.status} on URL: ${response.url}`);
	}

	if (response.status === 404) {
		errorStore.setError(404, "ページが見つかりません。");
		throw new Error(`Not found: ${response.url}`);
	}

	if (response.status === 401) {
		let logoutSuccessful = false;
		try {
			const res = await fetchFunction(`${PUBLIC_API_URL}/auth/logout`, {
				method: "POST",
				credentials: "include"
			});

			if (!res.ok) {
				errorStore.setError(500, "処理中に予期せぬエラーが発生しました。");
			} else {
				logoutSuccessful = true;
			}
		} catch (logoutError) {
			console.error("Logout request failed:", logoutError);
			errorStore.setError(500, "処理中に予期せぬエラーが発生しました。");
		}

		const currentErrorState = get(errorStore);
		if (!currentErrorState.statusCode && logoutSuccessful) {
			goto("/auth/login");
		}

		throw new Error("Session expired or unauthorized.");
	}

	return response;
}
