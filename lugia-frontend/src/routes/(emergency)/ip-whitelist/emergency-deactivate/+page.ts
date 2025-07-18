import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ url, fetch }) => {
	// TODO: check if this works as intended. probablly not.
	const token = url.searchParams.get("token");

	if (!token) {
		throw error(400, "この緊急解除リンクは無効です。新しい緊急解除リンクを取得してください。");
	}

	try {
		const response = await fetch(
			`/api/ip-whitelist/emergency-deactivate?token=${encodeURIComponent(token)}`,
			{
				method: "POST"
			}
		);

		if (!response.ok) {
			if (response.status === 400) {
				throw error(
					400,
					"この緊急解除リンクは無効か期限切れです。新しい緊急解除リンクを取得してください。"
				);
			} else if (response.status === 404) {
				throw error(404, "IP制限が見つかりません。既に解除されている可能性があります。");
			} else {
				throw error(
					response.status,
					"サーバーとの通信中に問題が発生しました。お手数ですが、時間をおいて再度お試しください。"
				);
			}
		}

		return {
			success: true,
			message: "IP制限が正常に解除されました。"
		};
	} catch (e) {
		// checking if error is a SvelteKit error
		if (
			e &&
			typeof e === "object" &&
			"status" in e &&
			typeof e.status === "number" &&
			"body" in e
		) {
			const body = e.body as { message?: string };
			if (!body.message) {
				throw error(
					503,
					"サーバーとの通信中に問題が発生しました。お手数ですが、時間をおいて再度お試しください。"
				);
			}

			// Re-throw SvelteKit errors explicitly
			throw error(e.status, body.message);
		}
		console.error("Error in emergency deactivate load function:", e);
		throw error(
			503,
			"サーバーとの通信中に問題が発生しました。お手数ですが、時間をおいて再度お試しください。"
		);
	}
};
