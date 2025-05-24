import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import { PUBLIC_API_URL } from "$env/static/public";

export const load: PageLoad = async ({ url, fetch }) => {
	const token = url.searchParams.get("token");

	if (!token) {
		throw error(
			400,
			"このパスワードリセットリンクは無効か期限切れです。お手数ですが、再度リセットをリクエストしてください。"
		);
	}

	try {
		const response = await fetch(`${PUBLIC_API_URL}/auth/verify-reset-token`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json"
			},
			body: JSON.stringify({ token })
		});

		if (!response.ok) {
			if (response.status === 400) {
				throw error(
					400,
					"このパスワードリセットリンクは無効か期限切れです。お手数ですが、再度リセットをリクエストしてください。"
				);
			} else {
				throw error(
					response.status,
					"サーバーとの通信中に問題が発生しました。お手数ですが、時間をおいて再度お試しください。"
				);
			}
		}

		const result = await response.json();

		return {
			token,
			email: result.email as string
		};
	} catch (e) {
		if (
			e &&
			typeof e === "object" &&
			"status" in e &&
			typeof e.status === "number" &&
			"message" in e &&
			typeof e.message === "string"
		) {
			// Re-throw SvelteKit errors explicitly
			throw error(e.status, e.message);
		}
		console.error("Error in /auth/reset-password load function:", e);
		throw error(
			503,
			"サーバーとの通信中に問題が発生しました。お手数ですが、時間をおいて再度お試しください。"
		);
	}
};
