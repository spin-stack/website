// @ts-check
import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';
import { rehypeBaseLinks } from './src/lib/base-links.mjs';
import { rehypeFeatureLinks } from './src/lib/feature-links.mjs';
import starlightImageZoom from 'starlight-image-zoom';
import starlightSidebarTopics from 'starlight-sidebar-topics';
import { sidebarTopics } from './src/lib/nav.mjs';

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
			title: 'Spinbox',
			description:
				'Self-hosted sandboxes in your own AWS account: a virtual machine per workspace, snapshots with memory, rollback, and a diff of what changed.',
			// The organization's own mark (spin-stack/brand): the tile on its navy in dark, and clear in light,
			// where the tile would read as a sticker.
			logo: { dark: './src/assets/logo.png', light: './src/assets/logo-light.png', alt: 'Spinbox' },
			favicon: '/favicon.png',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/spin-stack' }],
			customCss: ['./src/styles/brand-tokens.css', './src/styles/brand-starlight.css', './src/styles/custom.css'],
			// A hand-written page is edited here; a generated one names its source instead (Footer.astro).
			editLink: { baseUrl: 'https://github.com/spin-stack/website/edit/main/' },
			lastUpdated: true,
			components: {
				Banner: './src/components/Banner.astro',
				Hero: './src/components/Hero.astro',
				Head: './src/components/Head.astro',
				Footer: './src/components/Footer.astro',
				PageTitle: './src/components/PageTitle.astro',
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
			plugins: [
				// The docs, installing, security and the reference each have a sidebar of their own.
				starlightSidebarTopics(sidebarTopics(), { exclude: ['/'] }),
				// A diagram or a screenshot opens full size in place (Figure.astro emits its markup).
				starlightImageZoom(),
			],
		}),
	],
});
