// node --test: a post link resolves to the post on the blog, and a slug the blog has not
// published fails. The posts here are made up; posts.json holds the blog's.
import assert from 'node:assert/strict';
import { test } from 'node:test';
import published from '../data/posts.json' with { type: 'json' };
import { postHref } from './post-links.mjs';

const record = { blog: 'https://blog.example', posts: [{ slug: 'a-post', path: '/spinbox/a-post/', title: 'A post', date: '2026-11-24' }] };

test('a published post is its address on the blog', () => {
	assert.deepEqual(postHref('a-post', record), { href: 'https://blog.example/spinbox/a-post/', post: record.posts[0] });
});

for (const [name, slug] of [
	['a slug the blog has not published', 'a-draft'],
	['a slug written with its path', '/spinbox/a-post/'],
	['no slug', ''],
]) {
	test(`${name} fails`, () => assert.throws(() => postHref(slug, record), /has published no post/));
}

test('posts.json is a record the links can read', () => {
	assert.match(published.blog, /^https:\/\/[^/]+$/);
	for (const p of published.posts) {
		assert.match(p.path, new RegExp(`^/[a-z0-9-]+/${p.slug}/$`));
		assert.equal(postHref(p.slug).href, `${published.blog}${p.path}`);
	}
});
