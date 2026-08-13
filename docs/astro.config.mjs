// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightThemeTerminal from 'starlight-theme-terminal';

// https://astro.build/config
export default defineConfig({
	site: 'https://aliq80.github.io',
	base: '/jutsu',
	integrations: [
		starlight({
			title: 'Jutsu',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/AliQ80/jutsu' }],
			plugins: [starlightThemeTerminal()],
			sidebar: [
				{
					label: 'Getting Started',
					items: [{ label: 'Installation', slug: 'installation' }],
				},
				{
					label: 'Usage',
					items: [
						{ label: 'Overview', slug: 'usage' },
						{ label: 'Interface', slug: 'usage/interface' },
						{ label: 'Navigation', slug: 'usage/navigation' },
						{ label: 'Commands', slug: 'usage/commands' },
						{ label: 'Flags', slug: 'usage/flags' },
						{ label: 'Inputs', slug: 'usage/inputs' },
						{ label: 'Global Options', slug: 'usage/global-options' },
						{ label: 'Running Commands', slug: 'usage/execution' },
						{ label: 'Output & Docs', slug: 'usage/output-and-docs' },
					],
				},
				{
					label: 'Workflows & Tutorials',
					items: [
						{ label: 'Composing a Command', slug: 'workflows/composing-a-command' },
						{ label: 'Inputs & Global Options', slug: 'workflows/inputs-and-global-options' },
						{ label: 'Interactive & Remote Operations', slug: 'workflows/interactive-and-remote-operations' },
					],
				},
			],
		}),
	],
});
