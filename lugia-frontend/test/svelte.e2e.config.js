import adapter from "@sveltejs/adapter-node";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),

	kit: {
		adapter: adapter(),
		alias: {
			$components: "src/components"
		},
		// For E2E testing, disabling CSRF origin checking can simplify requests
		// between containers or from test runners that might not send standard browser origins.
		csrf: {
			checkOrigin: false
		}
	}
};

export default config;
