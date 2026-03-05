/** @type {import('eslint').ESLint.Plugin} */
const plugin = {
	rules: {
		"enforce-absolute-imports": {
			meta: {
				type: "problem",
				schema: [
					{
						type: "object",
						properties: {
							service: { type: "string" },
						},
						required: ["service"],
						additionalProperties: false,
					},
				],
				docs: {
					description:
						"Enforce absolute imports using service aliases ($lugia/, $giratina/) instead of relative paths or $lib/",
				},
				messages: {
					noRelativeImport:
						'Use {{alias}} instead of relative import "{{source}}".',
					noLibAlias:
						"Use {{alias}}/lib/... instead of $lib/. The $lib alias is banned to enforce one canonical import path per file.",
					noZoroarkBarrel:
						'Use deep imports like "@dislyze/zoroark/Button" instead of barrel import "@dislyze/zoroark". Each component/utility has its own subpath.',
				},
			},
			create(context) {
				const service = context.options[0]?.service;

				function getAlias() {
					const filename = context.filename || context.getFilename();
					// Determine if file is in test/ or src/ based on path
					if (filename.includes("/test/")) {
						return `$${service}-test`;
					}
					return `$${service}`;
				}

				function check(node) {
					const source = node.source?.value;
					if (!source) return;

					// SvelteKit auto-generated types — no alternative exists
					if (source === "./$types") return;

					if (source.startsWith("./") || source.startsWith("../")) {
						context.report({
							node: node.source,
							messageId: "noRelativeImport",
							data: { source, alias: `${getAlias()}/` },
						});
					}

					if (source.startsWith("$lib/") || source === "$lib") {
						context.report({
							node: node.source,
							messageId: "noLibAlias",
							data: { alias: `$${service}` },
						});
					}

					if (source === "@dislyze/zoroark") {
						context.report({
							node: node.source,
							messageId: "noZoroarkBarrel",
						});
					}
				}

				return {
					ImportDeclaration: check,
					// Also catch `export { x } from "./foo"`
					ExportNamedDeclaration: check,
					ExportAllDeclaration: check,
				};
			},
		},
	},
};

export default plugin;
