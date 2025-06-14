import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		port: 4000,
		proxy: {
			"/api": {
				target: "http://localhost:4001",
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
