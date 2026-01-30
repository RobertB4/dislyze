import type { PageLoad } from "./$types";
import { loadFunctionFetch } from "$lib/fetch";
import { type EnterpriseFeatures as BaseEnterpriseFeatures } from "@dislyze/zoroark";

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

export const load: PageLoad = async ({ fetch, parent }) => {
	const { me } = await parent();
	const tenantsPromise = loadFunctionFetch(fetch, "/api/tenants").then(
		(res) => res.json() as Promise<GetTenantsResponse>
	);

	return {
		me,
		tenantsPromise
	};
};
