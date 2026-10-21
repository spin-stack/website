// @ts-check
import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';
import { rehypeBaseLinks } from './src/lib/base-links.mjs';
import { rehypeFeatureLinks } from './src/lib/feature-links.mjs';
import { sidebar } from './src/lib/nav.mjs';

// Where the site is served: a GitHub project page is https://<org>.github.io/<repo>/. The
// content links from the root, and rehypeBaseLinks puts the base in front.
const site = process.env.SITE_URL || 'https://spin-stack.github.io';
const base = process.env.BASE_PATH || '/website';

export default defineConfig({
	site,
	base,
	trailingSlash: 'always',
	markdown: { rehypePlugins: [rehypeFeatureLinks, [rehypeBaseLinks, { base }]] },
	integrations: [
		starlight({
			title: 'Spin',
			description:
				'Self-hosted sandboxes in your own AWS account: a virtual machine per workspace, snapshots with memory, rollback, and a diff of what changed.',
			logo: { src: './src/assets/logo.svg' },
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/spin-stack' }],
			customCss: ['./src/styles/custom.css'],
			components: {
				Banner: './src/components/Banner.astro',
				Hero: './src/components/Hero.astro',
				Head: './src/components/Head.astro',
				Footer: './src/components/Footer.astro',
			},
			head: [
				// The link preview a chat shows: public/og.png, by its absolute address.
				{ tag: 'meta', attrs: { property: 'og:image', content: `${site}${base.replace(/\/$/, '')}/og.png` } },
				{ tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
				{
					tag: 'link',
					attrs: { rel: 'alternate', type: 'text/plain', title: 'llms.txt', href: `${base.replace(/\/$/, '')}/llms.txt` },
				},
			],
			sidebar: sidebar(),
		}),
	],
});
