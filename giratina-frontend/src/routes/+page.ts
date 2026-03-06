// Feature doc: docs/features/tenant-onboarding.md, docs/features/tenant-impersonation.md
import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$giratina/lib/fetch";
import { type EnterpriseFeatures as BaseEnterpriseFeatures } from "@dislyze/zoroark/meCache";

// Extended type for giratina internal app - includes SSO data not exposed to public apps
export type EnterpriseFeatures = BaseEnterpriseFeatures & {
	sso: {
		enabled: boolean;
		idp_metadata_url: string;
		attribute_mapping: Record<string, string>;
		allowed_domains: string[];
	};
};

export interface Tenant {
	id: string;
	name: string;
	enterprise_features: EnterpriseFeatures;
	stripe_customer_id?: string;
	created_at: string;
	updated_at: string;
}

export interface GetTenantsResponse {
	tenants: Tenant[];
}

export async function load({ fetch, parent }: Parameters<PageLoad>[0]) {
	const { me } = await parent();
	const tenantsPromise = loadFunctionFetch(fetch, "/api/tenants").then(
		(res) => res.json() as Promise<GetTenantsResponse>
	);

	return {
		me,
		tenantsPromise
	};
}
