import { API_URL } from "$env/static/private";
import type { Handle } from "@sveltejs/kit";
import { dev } from "$app/environment";

function buildCookieString(name: string, value: string, options: any): string {
	let parts = [`${name}=${encodeURIComponent(value)}`];
	if (options.path) parts.push(`Path=${options.path}`);
	if (options.domain) parts.push(`Domain=${options.domain}`);
	if (options.maxAge !== undefined) parts.push(`Max-Age=${options.maxAge}`);
	if (options.expires) parts.push(`Expires=${options.expires.toUTCString()}`);
	if (options.httpOnly) parts.push("HttpOnly");
	if (options.secure) parts.push("Secure");
	if (options.sameSite) parts.push(`SameSite=${options.sameSite}`); // Assumes options.sameSite is already correctly cased e.g. Lax, Strict, None
	return parts.join("; ");
}

export const handle: Handle = async ({ event, resolve }) => {
	if (event.url.pathname.startsWith("/api/")) {
		// Prevent SvelteKit from trying to route this internally
		event.setHeaders({ "x-sveltekit-normalize": "false" });

		const targetPath = event.url.pathname.substring("/api".length);
		const targetURL = new URL(targetPath + event.url.search, API_URL);

		// Clean up headers for the backend request
		const requestHeaders = new Headers(event.request.headers);
		requestHeaders.delete("connection"); // Let node handle connection pooling
		requestHeaders.delete("host"); // Avoid host mismatch

		// Forward cookies from the client to the backend
		const clientCookies = event.cookies.getAll();
		if (clientCookies.length > 0) {
			requestHeaders.set("cookie", clientCookies.map((c) => `${c.name}=${c.value}`).join("; "));
		} else {
			requestHeaders.delete("cookie");
		}

		const clientIp = event.getClientAddress();
		if (clientIp) {
			requestHeaders.set("X-Forwarded-For", clientIp);
		}

		console.log(`[HOOK] Proxying request from ${event.url.pathname} to ${targetURL.toString()}`);
		try {
			const fetchOptions: RequestInit = {
				method: event.request.method,
				headers: requestHeaders,
				redirect: "manual" // Prevent Node fetch from following redirects automatically
			};

			if (event.request.method !== "GET" && event.request.method !== "HEAD") {
				fetchOptions.body = await event.request.blob();
			}
			const backendResponse = await fetch(targetURL.toString(), fetchOptions);

			const finalResponseHeaders = new Headers(backendResponse.headers);
			// Remove headers that should not be directly proxied or are handled differently
			finalResponseHeaders.delete("content-encoding"); // Avoid double compression
			finalResponseHeaders.delete("transfer-encoding"); // Avoid issues with chunked encoding if SvelteKit/host handles it
			finalResponseHeaders.delete("set-cookie"); // Clear any original Set-Cookie headers from backend, we will re-process them.

			const backendSetCookieHeaders = backendResponse.headers.getSetCookie();
			if (backendSetCookieHeaders && backendSetCookieHeaders.length > 0) {
				backendSetCookieHeaders.forEach((cookieString) => {
					const parts = cookieString.split(";").map((p) => p.trim());
					const [nameValue, ...attrs] = parts;
					const [name, ...valueParts] = nameValue.split("=");
					const value = valueParts.join("=");

					const options: {
						path: string;
						httpOnly: boolean;
						maxAge?: number;
						sameSite?: "strict" | "lax" | "none";
						secure: boolean;
						domain?: string;
						expires?: Date;
					} = {
						path: "/",
						httpOnly: false,
						secure: !dev
					};

					attrs.forEach((attr) => {
						const [attrName, ...attrValueParts] = attr.split("=");
						const attrValue = attrValueParts.join("=");
						const lowerAttrName = attrName.toLowerCase();

						if (lowerAttrName === "path") options.path = attrValue;
						else if (lowerAttrName === "httponly") options.httpOnly = true;
						else if (lowerAttrName === "max-age") {
							const maxAgeNum = parseInt(attrValue, 10);
							if (!isNaN(maxAgeNum)) options.maxAge = maxAgeNum;
						} else if (lowerAttrName === "samesite") {
							const lowerAttrValue = attrValue.toLowerCase();
							if (lowerAttrValue === "strict") options.sameSite = "strict";
							else if (lowerAttrValue === "lax") options.sameSite = "lax";
							else if (lowerAttrValue === "none") options.sameSite = "none";
						} else if (lowerAttrName === "domain") options.domain = attrValue;
						else if (lowerAttrName === "expires") options.expires = new Date(attrValue);
					});

					if (API_URL.startsWith("http://")) {
						options.secure = false;
					} else {
						const backendSentSecure = attrs.some((attr) => attr.toLowerCase().startsWith("secure"));
						options.secure = backendSentSecure && !API_URL.startsWith("http://") ? true : !dev;
					}

					if (name) {
						// Still call event.cookies.set for SvelteKit's internal consistency / other potential uses
						event.cookies.set(name, value, options);

						// Manually add the processed cookie to the headers we will send to the client
						const clientCookieString = buildCookieString(name, value, options);
						finalResponseHeaders.append("Set-Cookie", clientCookieString);
						console.log(
							`[HOOK] Setting cookie for client: ${name} with options:`,
							JSON.stringify(options, null, 2)
						);
					}
				});
			}

			return new Response(backendResponse.body, {
				status: backendResponse.status,
				statusText: backendResponse.statusText,
				headers: finalResponseHeaders // Use these headers for the client response
			});
		} catch (error) {
			console.error("[HOOK] Proxy error:", error);
			let errorMessage = "API Proxy Error";
			let errorStatus = 502;
			if (error instanceof Error) {
				errorMessage = error.message;
				if (error.message.includes("ECONNREFUSED")) {
					errorMessage = `API Connection Refused at ${API_URL}`;
					errorStatus = 503;
				}
			}
			return new Response(errorMessage, { status: errorStatus });
		}
	}

	return resolve(event);
};
