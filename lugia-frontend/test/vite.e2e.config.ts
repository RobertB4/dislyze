import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
	root: ".",
	appType: "spa",
	plugins: [tailwindcss(), sveltekit()],
	preview: {
		host: "0.0.0.0",
		port: 23000,
		strictPort: true,
		cors: true
	}
});
