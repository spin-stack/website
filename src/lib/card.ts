import { readFileSync } from 'node:fs';
import satori from 'satori';
import sharp from 'sharp';

// A page's social card - the picture a chat or a feed shows beside its link - drawn from what the
// page already has: its topic, its title and its description. Nobody draws one by hand, so a new
// page, a generated one included, has its card the build it appears.
//
// satori lays the text out and draws each glyph as a path, so the picture needs no font installed
// where the site is built; sharp, which Astro already takes, turns it into a PNG.

const W = 1200;
const H = 630;
const PAD = 72;

// The colours are brand's, as task reference copies them: the dark panel's tokens, the site's
// dark theme.
const tokens = readFileSync('src/styles/brand-tokens.css', 'utf8');
function token(name: string): string {
	const hex = tokens.match(new RegExp(`--spin-${name}:\\s*(#[0-9A-Fa-f]{6})`))?.[1];
	if (!hex) throw new Error(`src/styles/brand-tokens.css has no --spin-${name}`);
	return hex;
}
const c = { page: token('page'), text: token('text'), muted: token('muted'), accent: token('accent-text'), dim: token('border-strong') };

// IBM Plex as the site serves it. satori reads WOFF, not the WOFF2 the pages load.
const face = (pkg: string, weight: 400 | 600) => readFileSync(`node_modules/@fontsource/${pkg}/files/${pkg}-latin-${weight}-normal.woff`);
const fonts = [
	{ name: 'Plex Sans', data: face('ibm-plex-sans', 400), weight: 400 as const },
	{ name: 'Plex Sans', data: face('ibm-plex-sans', 600), weight: 600 as const },
	{ name: 'Plex Mono', data: face('ibm-plex-mono', 600), weight: 600 as const },
];
// The organization's mark drawn for the navy page, as the header has it.
const mark = `data:image/png;base64,${readFileSync('src/assets/logo.png').toString('base64')}`;

type Node = { type: string; props: Record<string, unknown> };
const el = (type: string, style: Record<string, unknown>, children?: unknown, extra = {}): Node => ({
	type,
	props: { style, children, ...extra },
});

export interface Card {
	topic: string;
	title: string;
	description: string;
	/** Where the site is served, without the scheme: docs.spinbox.dev, or a project page's host and path. */
	site: string;
}

export async function card({ topic, title, description, site }: Card): Promise<Buffer> {
	const mono = { fontFamily: 'Plex Mono', fontWeight: 600, letterSpacing: 1.5 };
	const column = W - 2 * PAD;
	// The column has its size written out: satori grows a padded box by its padding, and the
	// last line fell off the card.
	const tree = el('div', { width: W, height: H, display: 'flex', background: c.page, fontFamily: 'Plex Sans' }, [
		el('div', { width: column, height: H - 2 * PAD, margin: PAD, display: 'flex', flexDirection: 'column' }, [
			el('div', { display: 'flex', alignItems: 'center', fontSize: 22, ...mono }, [
				el('img', { width: 64, height: 64, marginRight: 20 }, undefined, { src: mark, width: 64, height: 64 }),
				el('span', { color: c.accent }, 'SPINBOX'),
				el('span', { color: c.muted, marginLeft: 16 }, `·  ${topic.toUpperCase()}`),
			]),
			el('div', { flexGrow: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center' }, [
				el('div', { width: column, fontSize: title.length > 32 ? 64 : 76, fontWeight: 600, lineHeight: 1.1, color: c.text, letterSpacing: -1 }, title),
				el('div', { width: column, marginTop: 28, fontSize: 30, lineHeight: 1.4, color: c.muted, display: 'block', lineClamp: 3 }, description),
			]),
			el('div', { alignSelf: 'flex-end', fontSize: 18, color: c.dim, ...mono, letterSpacing: 0 }, site),
		]),
	]);
	const svg = await satori(tree as never, { width: W, height: H, fonts });
	return sharp(Buffer.from(svg)).png().toBuffer();
}
