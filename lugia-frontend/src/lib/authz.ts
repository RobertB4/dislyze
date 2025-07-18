import type { Me, EnterpriseFeatures } from "@dislyze/zoroark";

export function hasPermission(
	me: Me,
	permission: `${"tenant" | "users" | "roles" | "ip_whitelist"}.${"view" | "edit"}`
): boolean {
	if (me.permissions.includes(permission)) {
		return true;
	}

	if (permission.endsWith(".view")) {
		const editPermission = permission.replace(".view", ".edit") as typeof permission;
		return me.permissions.includes(editPermission);
	}

	return false;
}

export function hasFeature(me: Me, feature: keyof EnterpriseFeatures): boolean {
	return me.enterprise_features[feature].enabled;
}
