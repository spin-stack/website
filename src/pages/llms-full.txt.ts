import type { APIRoute } from 'astro';
import { home, pagePath, sections, siteBase, toMarkdown } from '../lib/pages';

// Every page's Markdown, in sidebar order, each headed by where it lives on the site.
export const GET: APIRoute = async ({ site }) => {
	const base = siteBase(site);
	const parts = [toMarkdown(await home(), base)];
	for (const s of await sections()) {
		for (const p of s.pages) parts.push(`<!-- ${base}${pagePath(p)} -->\n\n${toMarkdown(p, base)}`);
	}
	return new Response(parts.join('\n---\n\n'), { headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
};
