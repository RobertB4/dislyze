import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
	const redirectTo = url.searchParams.get("redirect");
	const message = url.searchParams.get("message");

	// Validate redirect URL for security (prevent open redirect attacks)
	let validatedRedirect = "/";
	if (redirectTo) {
		try {
			// Only allow internal URLs (same origin)
			const redirectUrl = new URL(redirectTo, url.origin);
			if (redirectUrl.origin === url.origin) {
				validatedRedirect = redirectUrl.pathname + redirectUrl.search;
			}
		} catch {
			// Invalid URL format, use default
			validatedRedirect = "/";
		}
	}

	return {
		redirectTo: validatedRedirect,
		message
	};
};