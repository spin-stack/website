// A claim a page makes about what Spin does is a link to the feature board's row that says so:
// [idle suspend](<feature:Idle suspend, with a question before it that a keep answers>). This
// points it at the row's area on the status page, with the row's proof as its title, and fails
// the build on a row the board does not have. refgen checks the same, and that the row is built.
import { visit } from 'unist-util-visit';
import board from '../data/features.json' with { type: 'json' };

const rows = new Map(board.map((r) => [r.name, r]));

// The id Starlight gives an area's heading: its words, lowercase, joined by hyphens. linkcheck
// fails a link to an id the page does not have.
const slug = (s) => s.toLowerCase().replace(/[^a-z0-9\s-]/g, '').trim().replace(/\s+/g, '-');

/** The status page's link for a feature: link, as a page writes it. */
export function featureHref(name) {
	const row = rows.get(name);
	if (!row) throw new Error(`feature: no row of spin's feature board is ${JSON.stringify(name)}`);
	return { href: `/reference/status/#${slug(row.area)}`, row };
}

export function rehypeFeatureLinks() {
	return (tree) => {
		visit(tree, 'element', (node) => {
			const href = node.tagName === 'a' ? node.properties?.href : undefined;
			if (typeof href !== 'string' || !href.startsWith('feature:')) return;
			const { href: to, row } = featureHref(decodeURIComponent(href.slice('feature:'.length)));
			node.properties.href = to;
			node.properties.title = `${row.name}: ${row.state}, ${row.proof}`;
			node.properties.dataFeature = row.state;
		});
	};
}
