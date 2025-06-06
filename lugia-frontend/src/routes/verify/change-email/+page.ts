import { error, redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ url, fetch }) => {
	const token = url.searchParams.get("token");

	if (!token) {
		error(400, "Missing verification token");
	}

	// Try to verify the email change
	let response;
	try {
		response = await fetch(`/api/me/verify-change-email?token=${encodeURIComponent(token)}`, {
			method: "GET"
		});
	} catch (err) {
		console.error("Email verification fetch error:", err);
		return {
			token,
			needsLogin: false,
			verificationFailed: true
		};
	}

	if (response.ok) {
		// Email verification successful, redirect to profile with success message
		redirect(302, "/settings/profile?email-verified=true");
	} else if (response.status === 401) {
		// User not authenticated, return data to show login prompt
		return {
			token,
			needsLogin: true,
			verificationFailed: false
		};
	} else {
		// Verification failed for other reasons
		return {
			token,
			needsLogin: false,
			verificationFailed: true
		};
	}
};
