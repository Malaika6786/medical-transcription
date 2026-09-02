import { defineConfig, loadEnv } from 'vite';
import vue from '@vitejs/plugin-vue';
import vuetify from 'vite-plugin-vuetify';
import path from 'path';
// https://vitejs.dev/config/
export default defineConfig(function (_a) {
    var mode = _a.mode;
    var env = loadEnv(mode, process.cwd(), '');
    return {
        plugins: [
            vue({
                template: {
                    compilerOptions: {
                        isCustomElement: function (tag) { return tag === 'corti-embedded'; }
                    }
                }
            }),
            vuetify({ autoImport: true }),
        ],
        resolve: {
            alias: {
                '@': path.resolve(__dirname, './src')
            }
        },
        server: {
            port: 5173,
            proxy: {
                '/api': {
                    target: env.VITE_BACKEND_URL || 'http://localhost:8080',
                    changeOrigin: true,
                    ws: true,
                }
            }
        }
    };
});
