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
		cors: true,
		allowedHosts: ["lugia-frontend"],
		proxy: {
			"/api": {
				target: "http://lugia-backend:23001",
				changeOrigin: true,
				secure: false,
				configure: (proxy) => {
					proxy.on("proxyReq", (proxyReq, req) => {
						if (req.socket.remoteAddress) {
							proxyReq.setHeader("X-Forwarded-For", req.socket.remoteAddress);
						}
					});
				}
			}
		}
	}
});
