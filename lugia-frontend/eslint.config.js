import prettier from 'eslint-config-prettier';
import js from '@eslint/js';
import { includeIgnoreFile } from '@eslint/compat';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import { fileURLToPath } from 'node:url';
import ts from 'typescript-eslint';
import dislyze from '../scripts/eslint-plugin-dislyze.js';
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
			'*.config.js',
			'*.config.ts',
			'test/**/*.config.js',
			'test/**/*.config.ts'
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
			'func-style': ['error', 'declaration'],
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
		rules: {
			'func-style': ['error', 'declaration'],
		}
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
			'@typescript-eslint/no-redundant-type-constituents': 'off',
			'@typescript-eslint/no-unsafe-return': 'off',
		}
	},
	{
		languageOptions: {
			globals: {
				...globals.browser,
				...globals.node
			}
		}
	},
	{
		files: ['src/**/*.ts', 'src/**/*.svelte'],
		plugins: { dislyze },
		rules: {
			'dislyze/enforce-absolute-imports': ['error', { service: 'lugia' }],
		}
	},
	{
		files: ['test/**/*.ts'],
		extends: [ts.configs.disableTypeChecked],
		plugins: { dislyze },
		rules: {
			'dislyze/enforce-absolute-imports': ['error', { service: 'lugia' }],
		}
	}
);
