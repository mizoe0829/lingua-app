import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const backendUrl = process.env.BACKEND_URL || 'http://localhost:8085';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5173,
		proxy: {
			'/api': {
				target: backendUrl,
				changeOrigin: true
			}
		}
	}
});

