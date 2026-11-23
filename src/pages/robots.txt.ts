import type { APIRoute } from 'astro';
import { siteBase } from '../lib/pages';

// What a crawler reads first: everything may be crawled, and where the sitemap and llms.txt are.
// Cloudflare puts its own content signals in front of this on docs.spinbox.dev.
export const GET: APIRoute = ({ site }) => {
	const base = siteBase(site);
	return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${base}/sitemap-index.xml\n# For language models: ${base}/llms.txt\n`, {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
