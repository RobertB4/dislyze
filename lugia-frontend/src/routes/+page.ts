import type { PageLoad } from "./$types";
import type { PageData } from "./types";

export const load: PageLoad<PageData> = () => {
	return {
		message: "hi"
	};
};
