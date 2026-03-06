import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export function load({ url }: Parameters<PageLoad>[0]) {
	const status = parseInt(url.searchParams.get("status") || "500", 10);
	const message = url.searchParams.get("message") || "処理中に予期せぬエラーが発生しました。";

	if (isNaN(status) || status < 400 || status > 599) {
		throw error(500, "エラーページに無効なステータスコードが渡されました。");
	}

	throw error(status, message);
}
