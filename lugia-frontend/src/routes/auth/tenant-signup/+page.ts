import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
	const token = url.searchParams.get("token") ?? "";

	let email = "";
	let companyName = "";
	let userName = "";
	
	if (token) {
		try {
			// Simple JWT decoding without verification (for display only)
			const payload = JSON.parse(atob(token.split(".")[1]));
			email = payload.email || "";
			companyName = payload.company_name || "";
			userName = payload.user_name || "";
		} catch (e) {
			console.error("Failed to decode token:", e);
		}
	}

	return {
		token,
		email,
		companyName,
		userName
	};
};
