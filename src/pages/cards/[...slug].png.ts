import { type CollectionEntry, getCollection } from 'astro:content';
import type { APIRoute, GetStaticPaths } from 'astro';
import { card } from '../../lib/card';
import { topicOf } from '../../lib/nav.mjs';
import { siteBase } from '../../lib/pages';

// Every page's social card but the landing page's, which is the hero's still (og.png.ts). Head.astro points
// og:image at it.
export const getStaticPaths: GetStaticPaths = async () =>
	(await getCollection('docs')).filter((d) => d.id !== 'index').map((d) => ({ params: { slug: d.id }, props: { doc: d } }));

export const GET: APIRoute<{ doc: CollectionEntry<'docs'> }> = async ({ props: { doc }, site }) =>
	new Response(
		new Uint8Array(
			await card({
				topic: topicOf(doc.id),
				title: doc.data.title,
				description: doc.data.description ?? '',
				site: siteBase(site).replace(/^https?:\/\//, ''),
			}),
		),
		{ headers: { 'Content-Type': 'image/png' } },
	);
