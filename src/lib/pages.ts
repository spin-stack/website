import { getCollection, type CollectionEntry } from 'astro:content';
import { experimentalMarkdown } from './experimental';
import { nav } from './nav.mjs';
import { statusLine } from './reference';

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
	return d.id === 'index' ? '/index.md' : `/${d.id}.md`;
}

// What a picture says, for a reader that gets no picture. A component used in the content and
// missing here fails the build, so a new diagram cannot vanish from the Markdown silently.
const figures: Record<string, string> = {
	Architecture:
		'An installation in one AWS account and VPC: users and agents reach the proxy on 443; the proxy ' +
		'forwards to the control plane on 8080; the control plane runs PostgreSQL on its own machine and ' +
		'archives it to an S3 bucket of its own; ' +
		'runners, an autoscaling group of 0..n machines each running many workspaces as KVM virtual ' +
		'machines, dial out to the control plane and to the proxy\'s relay; runners and the control plane ' +
		'read and write volumes in S3. Nothing opens a connection into a runner or a workspace.',
};

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
	body = body.replace(/<([A-Z]\w*)\s*\/>/g, (_, name: string) => {
		const text = figures[name];
		if (!text) throw new Error(`${d.id}: <${name} /> has no text in src/lib/pages.ts figures`);
		return `*Diagram: ${text}*`;
	});
	// MDX nests components with tabs; left in, Markdown reads the text inside them as code.
	body = body.replace(/^\t+/gm, '');
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
