// Feature doc: docs/features/tenant-onboarding.md, docs/features/tenant-impersonation.md
import type { PageLoad } from "./$types";
import { createLoadClient } from "$giratina/lib/api";

export async function load({ fetch, parent }: Parameters<PageLoad>[0]) {
	const { me } = await parent();
	const api = createLoadClient(fetch);
	const tenantsPromise = api.GET("/tenants").then(({ data }) => data!.tenants ?? []);

	return {
		me,
		tenantsPromise
	};
}
