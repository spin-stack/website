/** Where a page's Markdown twin is served, from the site's root: what src/pages/[...slug].md.ts writes. */
export function twinPath(id: string): string {
	return id === '' || id === 'index' ? '/index.md' : `/${id.replace(/\/$/, '')}.md`;
}
