import type { APIRoute, GetStaticPaths } from 'astro';
import { getCollection, type CollectionEntry } from 'astro:content';
import { siteBase, toMarkdown } from '../lib/pages';

// Each page's Markdown twin, at the page's path with .md: what llms.txt links to.
export const getStaticPaths: GetStaticPaths = async () =>
	(await getCollection('docs')).map((d) => ({ params: { slug: d.id }, props: { doc: d } }));

export const GET: APIRoute<{ doc: CollectionEntry<'docs'> }> = ({ props, site }) =>
	new Response(toMarkdown(props.doc, siteBase(site)), {
		headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
	});
