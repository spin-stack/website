import { getEntry } from 'astro:content';
import type { APIRoute } from 'astro';
import { landing } from '../lib/card';
import { siteBase } from '../lib/pages';
import { session } from '../lib/undo';

// The landing page's social card: the hero's title and the session its terminal types, drawn the
// build the hero changes. Head.astro points og:image at it.
export const GET: APIRoute = async ({ site }) => {
	const title = (await getEntry('docs', 'index'))?.data.hero?.title;
	if (!title) throw new Error('the landing page has no hero title');
	const cut = title.indexOf('. ');
	const plain = (s: string) => s.replaceAll('`', '');
	return new Response(
		new Uint8Array(
			await landing({
				lead: plain(cut < 0 ? title : title.slice(0, cut + 1)),
				rest: plain(cut < 0 ? '' : title.slice(cut + 2)),
				session,
				site: siteBase(site).replace(/^https?:\/\//, ''),
			}),
		),
		{ headers: { 'Content-Type': 'image/png' } },
	);
};
