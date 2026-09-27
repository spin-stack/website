// A link in the content is written from the site's root ("/concepts/network/"), and the site may
// be served under a path (a GitHub project page is /<repo>/). This puts the base in front of
// every root-relative link the content renders, so a page is written once for either.
import { visit } from 'unist-util-visit';

export function rehypeBaseLinks({ base }) {
	const prefix = base.replace(/\/$/, '');
	return (tree) => {
		if (!prefix) return;
		visit(tree, 'element', (node) => {
			const href = node.tagName === 'a' ? node.properties?.href : undefined;
			if (typeof href === 'string' && href.startsWith('/') && !href.startsWith('//') && !href.startsWith(`${prefix}/`)) {
				node.properties.href = prefix + href;
			}
		});
	};
}
