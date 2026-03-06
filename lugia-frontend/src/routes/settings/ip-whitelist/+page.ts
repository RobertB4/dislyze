import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lugia/lib/fetch";

export type IPWhitelistRule = {
	id: string;
	ip_address: string;
	label: string | null;
	created_by: string;
	created_at: string;
};

export type GetIPWhitelistResponse = IPWhitelistRule[];

export function load({ fetch }: Parameters<PageLoad>[0]) {
	const ipWhitelistPromise: Promise<GetIPWhitelistResponse> = loadFunctionFetch(
		fetch,
		`/api/ip-whitelist`
	).then((res) => res.json());

	return {
		ipWhitelistPromise
	};
}
