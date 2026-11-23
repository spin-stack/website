// A page that would explain why Spinbox is built a way links the blog post that does instead:
// [why](<post:the-slug>). This points it at the post on the blog, with its title as the link's,
// and fails the build on a slug the blog has not published: src/data/posts.json, which `task
// posts` writes from the blog's feed. A draft is private, and its slug may still change.
import { visit } from 'unist-util-visit';
import record from '../data/posts.json' with { type: 'json' };

/** The blog's address of a published post: slug, as a page writes it; record, posts.json. */
export function postHref(slug, { blog, posts } = record) {
	const post = posts.find((p) => p.slug === slug);
	if (!post) throw new Error(`post: the blog has published no post ${JSON.stringify(slug)} (src/data/posts.json; task posts reads the feed)`);
	return { href: `${blog}${post.path}`, post };
}

export function rehypePostLinks() {
	return (tree) => {
		visit(tree, 'element', (node) => {
			const href = node.tagName === 'a' ? node.properties?.href : undefined;
			if (typeof href !== 'string' || !href.startsWith('post:')) return;
			const { href: to, post } = postHref(href.slice('post:'.length));
			node.properties.href = to;
			node.properties.title = post.title;
		});
	};
}
