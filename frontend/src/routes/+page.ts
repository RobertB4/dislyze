import type { PageLoad } from "./$types";
import type { PageData } from "./types";
import { PUBLIC_API_URL } from "$env/static/public";
import { loadFunctionFetch } from "$lib/fetch";

export const load: PageLoad<PageData> = async () => {
	try {
		const response = await loadFunctionFetch(fetch, `${PUBLIC_API_URL}/health`);
		const data = await response.text();
		return {
			message: data
		};
	} catch (error) {
		console.error("Error fetching from backend:", error);
		return {
			message: "Error connecting to backend"
		};
	}
};
