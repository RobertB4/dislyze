import prettier from 'eslint-config-prettier';
import js from '@eslint/js';
import { includeIgnoreFile } from '@eslint/compat';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import { fileURLToPath } from 'node:url';
import ts from 'typescript-eslint';
const gitignorePath = fileURLToPath(new URL('../.gitignore', import.meta.url));

export default ts.config(
	includeIgnoreFile(gitignorePath),
	js.configs.recommended,
	...ts.configs.recommendedTypeChecked,
	{
		files: [
			'eslint.config.js',
			'vite.config.js',
			'svelte.config.js',
			'tailwind.config.js',
			'playwright.config.js',
			'vitest-setup-client.ts',
			'*.config.js',
			'*.config.ts'
		],
		extends: [ts.configs.disableTypeChecked],
		rules: {}
	},
	{
		files: ['src/**/*.ts'],
		languageOptions: {
			parser: ts.parser,
			parserOptions: {
				project: true,
				tsconfigRootDir: import.meta.dirname,
			},
		},
		rules: {
			'@typescript-eslint/no-unsafe-assignment': 'off',
			'@typescript-eslint/no-explicit-any': 'off',
			'@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],

		},
	},

	...svelte.configs.recommended,
	prettier,
	...svelte.configs.prettier,
	{
		files: ['**/*.svelte', 'src/**/*.svelte.ts', 'src/**/*.svelte.js'], 
		languageOptions: {
			parserOptions: {
				projectService: true, // For type-aware linting in <script lang="ts">
				extraFileExtensions: ['.svelte'],
				parser: ts.parser, 
			}
		},
		rules: {}
	},
	{
		// Global rule overrides
		rules: {
			'@typescript-eslint/only-throw-error': 'off',
			'@typescript-eslint/no-floating-promises': 'off',
			'@typescript-eslint/no-unsafe-assignment': 'off',
			'@typescript-eslint/no-unsafe-member-access': 'off',
			'@typescript-eslint/no-explicit-any': 'off',
			'@typescript-eslint/no-unsafe-call': 'off',
			'@typescript-eslint/no-unsafe-argument': 'off',
		}
	},
	{
		languageOptions: {
			globals: {
				...globals.browser,
				...globals.node
			}
		}
	}
);
