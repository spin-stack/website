import { getCollection, type CollectionEntry } from 'astro:content';
import { experimentalMarkdown } from './experimental';
import { featureHref } from './feature-links.mjs';
import { nav } from './nav.mjs';
import figures from '../data/figures.json';
import { output } from './output';
import { statusLine } from './reference';
import { storyMarkdown } from './story';
import { twinPath } from './twin';

type Doc = CollectionEntry<'docs'>;

export interface Section {
	label: string;
	pages: Doc[];
}

/** A directory's pages in the order the sidebar shows them: by `sidebar.order`, then by id. */
function inDirectory(docs: Doc[], directory: string): Doc[] {
	const order = (d: Doc) => d.data.sidebar?.order ?? Number.MAX_SAFE_INTEGER;
	return docs
		.filter((d) => d.id === directory || d.id.startsWith(`${directory}/`))
		.sort((a, b) => order(a) - order(b) || a.id.localeCompare(b.id));
}

/** Every page, grouped and ordered as the sidebar has them. */
export async function sections(): Promise<Section[]> {
	const docs = await getCollection('docs');
	const byId = new Map(docs.map((d) => [d.id, d]));
	const listed = new Set<string>(['index']);
	const out: Section[] = [];
	for (const g of nav) {
		const groups: Section[] = g.groups
			? g.groups.map(([label, directory]) => ({ label: `${g.label}: ${label}`, pages: inDirectory(docs, directory) }))
			: [
					{
						label: g.label,
						pages: g.items.map(([, slug]) => {
							const d = byId.get(slug);
							if (!d) throw new Error(`src/lib/nav.mjs names ${slug}, and there is no such page`);
							return d;
						}),
					},
				];
		for (const s of groups) for (const p of s.pages) listed.add(p.id);
		out.push(...groups);
	}
	// A page the sidebar does not reach is one nobody finds: refuse it rather than publish it.
	const orphans = docs.filter((d) => !listed.has(d.id)).map((d) => d.id);
	if (orphans.length) throw new Error(`pages not in src/lib/nav.mjs: ${orphans.join(', ')}`);
	return out;
}

export async function home(): Promise<Doc> {
	const d = (await getCollection('docs')).find((x) => x.id === 'index');
	if (!d) throw new Error('no index page');
	return d;
}

/** The path a page is served at, and the path of its Markdown twin. */
export function pagePath(d: Doc): string {
	return d.id === 'index' ? '/' : `/${d.id}/`;
}
export function mdPath(d: Doc): string {
	return twinPath(d.id);
}

/**
 * A page's source as Markdown a model can read: the MDX imports go, and the few components the
 * content uses become the Markdown they stand for.
 */
export function toMarkdown(d: Doc, base: string): string {
	let body = d.body ?? '';
	body = body.replace(/^import .*$\n?/gm, '');
	body = body.replace(/<CardGrid[^>]*>|<\/CardGrid>|<Steps>|<\/Steps>|<Tabs[^>]*>|<\/Tabs>/g, '');
	body = body.replace(/<LinkCard\s+([^>]*?)\/>/g, (_, attrs: string) => {
		const a = (k: string) => attrs.match(new RegExp(`${k}="([^"]*)"`))?.[1] ?? '';
		const href = a('href');
		return `- [${a('title')}](${href.startsWith('/') ? href : `/${href}`}): ${a('description')}`;
	});
	body = body.replace(/<Card\s+title="([^"]+)"[^>]*>/g, '**$1**');
	body = body.replace(/<\/Card>/g, '');
	body = body.replace(/<TabItem\s+label="([^"]+)"[^>]*>/g, '**$1**');
	body = body.replace(/<\/TabItem>/g, '');
	body = body.replace(/<Aside(?:\s+type="(\w+)")?(?:\s+title="([^"]+)")?\s*>/g, (_, type, title) =>
		`> **${title ?? (type ? type[0].toUpperCase() + type.slice(1) : 'Note')}:**`,
	);
	body = body.replace(/<\/Aside>/g, '');
	body = body.replace(/<StatusCounts\s*\/>/g, statusLine);
	// The landing page's parts: the story's steps with their commands, a card as its claim linked
	// to the row that proves it, the numbers as the sentence they come from, the last word's links.
	body = body.replace(/<Story\s*\/>/g, () => storyMarkdown());
	body = body.replace(/<Stats\s*\/>/g, statusLine);
	body = body.replace(/<\/?Bento>/g, '');
	body = body.replace(/<BentoCard\s+([^>]*?)>\n?/g, (_, attrs: string) => {
		const a = (k: string) => attrs.match(new RegExp(`${k}="([^"]*)"`))?.[1];
		const feature = a('feature');
		return `- **[${a('title')}](${feature ? featureHref(feature).href : a('href')}).** `;
	});
	body = body.replace(/\n?<\/BentoCard>/g, '');
	body = body.replace(/<Cta\s+title="([^"]*)"\s+links=\{([\s\S]*?)\}>\n?([\s\S]*?)\n?<\/Cta>/g, (_, title: string, list: string, text: string) => {
		const items = [...list.matchAll(/text:\s*'([^']+)',\s*href:\s*'([^']+)'/g)].map(([, t, h]) => `- [${t}](/${h.replace(/^\//, '')})`);
		return `**${title}** ${text.trim()}\n\n${items.join('\n')}`;
	});
	// A screenshot is its alt text: a twin has no picture to show.
	body = body.replace(/!\[([^\]]*)\]\([^)]*\.png\)/g, '*Screenshot: $1.*');
	// What a command prints is the same block of output.
	body = body.replace(
		/<Output\s+name="([\w.-]+)"\s+caption="([^"]*)"(?:\s+lang="(\w+)")?\s*\/>/g,
		(_, name: string, caption: string, lang = 'text') => `${caption}:\n\n\`\`\`${lang}\n${output(name)}\n\`\`\``,
	);
	// A diagram is its caption, its words in the order the picture has them, and where it is drawn.
	body = body.replace(/<Figure\s+name="([\w-]+)"\s+caption="([^"]*)"\s*\/>/g, (_, name: string, caption: string) => {
		const f = (figures as Record<string, { source: string; words: string[] }>)[name];
		if (!f) throw new Error(`${d.id}: <Figure name="${name}"> is no diagram in src/data/figures.json`);
		return `*Diagram: ${caption}* Its labels: ${f.words.join(' · ')}. ([source](${f.source}))`;
	});
	// MDX nests components with tabs; left in, Markdown reads the text inside them as code.
	body = body.replace(/^\t+/gm, '');
	// A claim's link to the feature board is a link to the row's area on the status page.
	body = body.replace(/\]\(<?feature:([^>)]+)>?\)/g, (_, name: string) => `](${featureHref(name).href})`);
	body = links(body, base);
	body = body.replace(/\n{3,}/g, '\n\n');
	untranslated(d, body);
	return `# ${d.data.title}\n\n${links(experimentalMarkdown, base)}\n\n> ${d.data.description}\n\n${body.trim()}\n`;
}

/** A link within the site is to the page's Markdown twin, absolute, so it works out of context. */
function links(md: string, base: string): string {
	return md.replace(/\]\((\/[^)\s#]*?)\/?(#[^)\s]*)?\)/g, (_, path: string, hash = '') =>
		`](${base}${path === '' ? '/index' : path}.md${hash})`,
	);
}

/**
 * untranslated refuses a twin that still carries MDX: a component, an expression or a directive
 * left in would reach a model as markup. Code blocks are left alone, since what they quote is
 * meant literally; braces are MDX only in an MDX page.
 */
function untranslated(d: Doc, body: string): void {
	const prose = body.replace(/^```[\s\S]*?^```/gm, '').replace(/`[^`\n]*`/g, '');
	const mdx = d.filePath?.endsWith('.mdx') ?? false;
	const left = prose.match(/<\/?[A-Z]\w*[\s/>]/) ?? prose.match(/^:{2,3}\w/m) ?? (mdx ? prose.match(/[{}]/) : null);
	if (left) throw new Error(`${d.id}: its Markdown twin still carries MDX: ${JSON.stringify(left[0])}`);
}

/** The site's address with its base and no trailing slash: what every absolute link starts with. */
export function siteBase(site: URL | undefined): string {
	return new URL(import.meta.env.BASE_URL, site ?? 'http://localhost').href.replace(/\/$/, '');
}
