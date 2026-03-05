import type { PageLoad } from "./$types";
import type { PageData } from "$lugia/routes/types";

export const load: PageLoad<PageData> = () => {
	return {
		message: "hi"
	};
};
