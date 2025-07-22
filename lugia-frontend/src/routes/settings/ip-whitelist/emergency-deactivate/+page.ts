import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

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

export const load: PageLoad = async ({ url, fetch }) => {
	const token = url.searchParams.get("token");

	if (!token) {
		return {
			error: "緊急解除リンクが無効です。サポートにお問い合わせください。"
		};
	}

	try {
		const response = await fetch(
			`/api/ip-whitelist/emergency-deactivate?token=${encodeURIComponent(token)}`,
			{
				method: "POST"
			}
		);

		if (!response.ok) {
			return {
				error: "緊急解除リンクが無効または期限切れです。サポートにお問い合わせください。"
			};
		}

		throw redirect(302, "/settings/ip-whitelist");
	} catch (e) {
		if (isRedirect(e)) throw e;

		console.error("Error in emergency deactivate load function:", e);
		return {
			error: "予期せぬエラーが発生しました。"
		};
	}
};
