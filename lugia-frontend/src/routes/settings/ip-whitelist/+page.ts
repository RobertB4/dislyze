// Feature doc: docs/features/ip-whitelisting.md
import type { PageLoad } from "./$types";
import { createLoadClient } from "$lugia/lib/api";

export function load({ fetch }: Parameters<PageLoad>[0]) {
	const api = createLoadClient(fetch);

	const ipWhitelistPromise = api.GET("/ip-whitelist").then(({ data }) => data!.rules);

	return {
		ipWhitelistPromise
	};
}
