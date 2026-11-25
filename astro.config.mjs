// @ts-check
import starlight from '@astrojs/starlight';
import { defineConfig, envField, sharpImageService } from 'astro/config';
import starlightImageZoom from 'starlight-image-zoom';
import starlightSidebarTopics from 'starlight-sidebar-topics';
import { rehypeBaseLinks } from './src/lib/base-links.mjs';
import { cspCheck, starlightInlineHashes } from './src/lib/csp.mjs';
import { rehypeFeatureLinks } from './src/lib/feature-links.mjs';
import { sidebarTopics } from './src/lib/nav.mjs';
import { rehypePostLinks } from './src/lib/post-links.mjs';

// Where the site is served: https://docs.spinbox.dev, at its root, and each preview at the root of its
// own pages.dev address. The content links from the root; under another base (BASE_PATH),
// rehypeBaseLinks puts it in front.
const site = process.env.SITE_URL || 'https://docs.spinbox.dev';
const base = process.env.BASE_PATH || '/';
// Google Tag Manager's container ID: given to production's build alone (pages.yml), and with none
// the site has no script of Google's and its policy names none of Google's addresses.
const ga = process.env.GTM_ID
	? {
			script: ['https://www.googletagmanager.com'],
			img: ' https://*.google-analytics.com https://*.googletagmanager.com',
			connect: ' https://*.google-analytics.com https://*.analytics.google.com https://*.googletagmanager.com',
		}
	: { script: [], img: '', connect: '' };

export default defineConfig({
	site,
	base,
	trailingSlash: 'always',
	// Screenshots re-encode at WebP 75: at Sharp's default (80) they weigh a fifth more, with no difference
	// visible at the width a page shows them.
	image: { service: sharpImageService({ webp: { quality: 75, effort: 6 } }) },
	// A Content-Security-Policy in a <meta> on every page: Astro hashes the scripts and styles it
	// emits, Starlight's inline scripts are hashed from its components (src/lib/csp.mjs), and
	// nothing else runs. Pagefind, the search, compiles WebAssembly. Cloudflare adds its Web
	// Analytics beacon at the edge, hence its script and the endpoint it reports to. Expressive
	// Code colours code and a figure sizes itself with style attributes, which a hash can't cover
	// and which can't run anything. What a <meta> can't say (frame-ancestors) is in public/_headers.
	security: {
		csp: {
			directives: [
				"default-src 'self'",
				`img-src 'self' data:${ga.img}`,
				`connect-src 'self' https://cloudflareinsights.com${ga.connect}`,
				"base-uri 'self'",
				"form-action 'self'",
				"object-src 'none'",
			],
			scriptDirective: {
				resources: ["'self'", "'wasm-unsafe-eval'", 'https://static.cloudflareinsights.com', ...ga.script],
				hashes: starlightInlineHashes(),
			},
			styleDirective: {
				// 'self' scoped to elements: an attribute has no address for it to match, and Astro warns when
				// style-src names it beside style-src-attr.
				resources: [
					{ resource: "'self'", kind: 'element' },
					{ resource: "'unsafe-inline'", kind: 'attribute' },
				],
			},
		},
	},
	// Every stylesheet a file, never a <style>: 'self' covers a file, and Astro can't hash the
	// styles of a component an MDX page imports.
	build: { inlineStylesheets: 'never' },
	// Nor is a face inlined as a data: URL, which the policy's default-src refuses: JetBrains Mono's
	// Greek and Vietnamese files are under Vite's 4 KiB limit, and each page logged a refused font.
	vite: { build: { assetsInlineLimit: (file) => (/\.woff2?$/.test(file) ? false : undefined) } },
	env: { schema: { GTM_ID: envField.string({ context: 'server', access: 'public', optional: true }) } },
	// Starlight colours code with Expressive Code; Astro's own Shiki colours nothing here, and with it on
	// Astro warns about CSP on every build.
	markdown: { syntaxHighlight: false, rehypePlugins: [rehypeFeatureLinks, rehypePostLinks, [rehypeBaseLinks, { base }]] },
	integrations: [
		cspCheck(),
		starlight({
			title: 'Spinbox',
			description: 'Workspaces in your own AWS account, each its own virtual machine, with snapshots that keep memory, rollback, and a diff of what changed.',
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
				PageSidebar: './src/components/PageSidebar.astro',
			},
			head: [
				// The link preview a chat shows is the page's card, which Head.astro names.
				{ tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
				// The mark iOS puts on a home screen.
				{ tag: 'link', attrs: { rel: 'apple-touch-icon', href: `${base.replace(/\/$/, '')}/apple-touch-icon.png` } },
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
