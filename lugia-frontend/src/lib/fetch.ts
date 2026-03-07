import { safeGoto } from "@dislyze/zoroark/routing";

/**
 * Handles errors from {#await} {:catch} blocks in pages.
 * Extracts status/message from the error and redirects to the error page.
 * Usage: {:catch e} {handleLoadError(e)} {/await}
 */
export function handleLoadError(e: unknown): never {
	console.error("Page data load error:", e);
	let status = 500;
	let message = "処理中に予期せぬエラーが発生しました。";

	const err = e as {
		status?: number;
		message?: string;
		body?: { message?: string };
		location?: string;
	};

	if (err.status) {
		status = err.status;
	}

	if (err.message) {
		message = err.message;
	}

	if (err.body?.message) {
		message = err.body.message;
	}

	if (e instanceof Error) {
		message = e.message;
	}

	if (err.location) {
		safeGoto(err.location);
	} else {
		safeGoto(`/error?status=${status}&message=${encodeURIComponent(message)}`);
	}

	throw e;
}
