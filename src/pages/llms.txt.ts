import type { APIRoute } from 'astro';
import { home, mdPath, sections, siteBase } from '../lib/pages';

// https://llmstxt.org: a title, a summary, and every page as a link to its Markdown with a line
// saying what it holds - the same order and grouping as the sidebar.
export const GET: APIRoute = async ({ site }) => {
	const base = siteBase(site);
	const idx = await home();
	const lines = [
		'# Spin',
		'',
		`> ${idx.data.description}`,
		'',
		'Every page is also served as Markdown at the linked `.md` path, and all of them together at ' +
			`${base}/llms-full.txt.`,
		'',
	];
	for (const s of await sections()) {
		lines.push(`## ${s.label}`, '');
		for (const p of s.pages) lines.push(`- [${p.data.title}](${base}${mdPath(p)}): ${p.data.description}`);
		lines.push('');
	}
	return new Response(lines.join('\n'), { headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
};
