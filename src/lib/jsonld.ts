// What a search engine or a model reads of a page besides its words: one JSON-LD graph a page
// (Head.astro). The person and the organization carry the blog's @ids (blog.spinbox.dev, its
// Base.astro), so both sites describe one author and one project. Every page has the site, the
// software it documents and its repositories; a page but the landing page has the way back to it
// and a TechArticle of its own.
import { nav, topics } from './nav.mjs';

const blog = 'https://blog.spinbox.dev/';

/** The public repositories the pages document, each with the language GitHub reads it as. */
const repositories = [
	{ name: 'spin', language: 'Go', what: 'The control plane, the runner, the proxy and the CLI.' },
	{ name: 'spin-terraform-aws', language: 'HCL', what: 'The OpenTofu module that installs Spinbox in an AWS account.' },
	{ name: 'machine', language: 'Go', what: 'QEMU, the guest kernel, the firmware and the base image a workspace boots, as one artefact.' },
];

export interface Page {
	id: string;
	title: string;
	description: string;
	lastUpdated?: Date;
	/** The page's card, by its absolute address. */
	image: string;
}

/**
 * The graph of a page at url, on the site at home (its root, base included). ids are every
 * page's id, so the trail links a reference group's index only where there is one.
 */
export function graph(page: Page, url: string, home: string, ids: Set<string>): object {
	const at = (path: string) => new URL(path.replace(/^\//, ''), home).href;
	const id = (name: string) => ({ '@id': `${home}#${name}` });
	const nodes: object[] = [
		{
			'@type': 'Person',
			'@id': `${blog}#person`,
			name: 'Manuel de Brito Fontes',
			url: `${blog}about/`,
			sameAs: ['https://github.com/aledbf', 'https://x.com/aledbf', 'https://www.linkedin.com/in/aledbf/'],
		},
		{
			'@type': 'Organization',
			'@id': `${blog}#organization`,
			name: 'Spinbox',
			alternateName: 'spin-stack',
			url: 'https://github.com/spin-stack',
			logo: at('/apple-touch-icon.png'),
			sameAs: [home, blog],
			founder: { '@id': `${blog}#person` },
		},
		{
			'@type': 'WebSite',
			...id('website'),
			name: 'Spinbox docs',
			url: home,
			inLanguage: 'en',
			publisher: { '@id': `${blog}#organization` },
		},
		{
			'@type': 'SoftwareApplication',
			...id('software'),
			name: 'Spinbox',
			applicationCategory: 'DeveloperApplication',
			operatingSystem: 'Linux',
			url: home,
			description: 'Workspaces in your own AWS account, each its own virtual machine, with snapshots that keep memory, rollback, and a diff of what changed.',
			license: 'https://www.apache.org/licenses/LICENSE-2.0',
			publisher: { '@id': `${blog}#organization` },
		},
		...repositories.map((r) => ({
			'@type': 'SoftwareSourceCode',
			'@id': `https://github.com/spin-stack/${r.name}`,
			name: r.name,
			description: r.what,
			codeRepository: `https://github.com/spin-stack/${r.name}`,
			programmingLanguage: r.language,
			license: 'https://www.apache.org/licenses/LICENSE-2.0',
			targetProduct: id('software'),
			publisher: { '@id': `${blog}#organization` },
		})),
	];
	if (page.id === '' || page.id === 'index' || page.id === '404') return { '@context': 'https://schema.org', '@graph': nodes };

	const steps = trail(page.id, ids);
	nodes.push({
		'@type': 'TechArticle',
		'@id': `${url}#article`,
		headline: page.title,
		description: page.description,
		url,
		mainEntityOfPage: url,
		image: page.image,
		...(page.lastUpdated ? { dateModified: page.lastUpdated.toISOString() } : {}),
		inLanguage: 'en',
		articleSection: steps[0]?.name,
		about: id('software'),
		isPartOf: id('website'),
		author: { '@id': `${blog}#person` },
		publisher: { '@id': `${blog}#organization` },
	});
	const crumbs = [{ name: 'Spinbox docs', path: '/' }, ...steps, { name: page.title, path: `/${page.id}/` }];
	nodes.push({
		'@type': 'BreadcrumbList',
		'@id': `${url}#breadcrumb`,
		itemListElement: crumbs.map((c, i) => ({ '@type': 'ListItem', position: i + 1, name: c.name, item: at(c.path) })),
	});
	return { '@context': 'https://schema.org', '@graph': nodes };
}

/**
 * The pages between the landing page and the page id, as its sidebar has them: its topic, and a
 * reference group's index where it has one. A step that is the page itself, or the same page as
 * the one before it, is left out ("Reference" is the CLI's index).
 */
function trail(id: string, ids: Set<string>): { name: string; path: string }[] {
	const group = nav.find((g) => (g.groups ? g.groups.some(([, dir]) => id === dir || id.startsWith(`${dir}/`)) : g.items.some(([, slug]) => slug === id)));
	const topic = topics.find((t) => t.id === group?.topic);
	if (!group || !topic) return [];
	const steps = [{ name: topic.label, path: topic.link }];
	const sub = group.groups?.find(([, dir]) => id.startsWith(`${dir}/`));
	if (sub && ids.has(sub[1])) steps.push({ name: sub[0], path: `/${sub[1]}/` });
	return steps.filter((s, i) => s.path !== `/${id}/` && s.path !== steps[i - 1]?.path);
}
