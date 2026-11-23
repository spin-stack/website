import { readFileSync } from 'node:fs';
import satori from 'satori';
import sharp from 'sharp';
import type { Line } from './undo';

// A page's social card - the picture a chat or a feed shows beside its link - drawn from what the
// page already has: its topic, its title and its description. Nobody draws one by hand, so a new
// page, a generated one included, has its card the build it appears.
//
// satori lays the text out and draws each glyph as a path, so the picture needs no font installed
// where the site is built; sharp, which Astro already takes, turns it into a PNG.

const W = 1200;
const H = 630;
const PAD = 72;

// The colours are brand's, as task reference copies them: a page's card in the dark panel's
// tokens, the landing page's in the light theme's.
const tokens = readFileSync('src/styles/brand-tokens.css', 'utf8');
// The light theme's block, which redefines the dark roles; the terminal's are in neither and the same in both.
const lightBlock = tokens.slice(tokens.indexOf(":root[data-theme='light']"));
function token(name: string, from = tokens): string {
	const hex = from.match(new RegExp(`--spin-${name}:\\s*(#[0-9A-Fa-f]{6})`))?.[1];
	if (!hex) throw new Error(`src/styles/brand-tokens.css has no --spin-${name}`);
	return hex;
}
const c = { page: token('page'), text: token('text'), muted: token('muted'), accent: token('accent-text'), dim: token('border-strong') };

// Inter and JetBrains Mono as the site serves them. satori reads WOFF, not the WOFF2 the pages load.
const face = (pkg: string, weight: 400 | 600) => readFileSync(`node_modules/@fontsource/${pkg}/files/${pkg}-latin-${weight}-normal.woff`);
const fonts = [
	{ name: 'Inter', data: face('inter', 400), weight: 400 as const },
	{ name: 'Inter', data: face('inter', 600), weight: 600 as const },
	{ name: 'JetBrains Mono', data: face('jetbrains-mono', 400), weight: 400 as const },
	{ name: 'JetBrains Mono', data: face('jetbrains-mono', 600), weight: 600 as const },
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
	const mono = { fontFamily: 'JetBrains Mono', fontWeight: 600, letterSpacing: 1.5 };
	const column = W - 2 * PAD;
	// The column has its size written out: satori grows a padded box by its padding, and the
	// last line fell off the card.
	const tree = el('div', { width: W, height: H, display: 'flex', background: c.page, fontFamily: 'Inter' }, [
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

export interface Landing {
	/** The hero's title: its first sentence, then the rest in the accent. */
	lead: string;
	rest: string;
	session: Line[];
	site: string;
}

// The landing page's card: the hero as the light theme draws it, the title beside the terminal.
// The terminal is black in either theme, as the palette has it.
export async function landing({ lead, rest, session, site }: Landing): Promise<Buffer> {
	const l = {
		page: token('page', lightBlock),
		text: token('text', lightBlock),
		muted: token('muted', lightBlock),
		accent: token('accent-text', lightBlock),
		border: token('border', lightBlock),
	};
	const t = {
		page: token('terminal-background'),
		text: token('terminal-foreground'),
		muted: token('terminal-bright-black'),
		ok: token('terminal-green'),
		bad: token('terminal-red'),
		edge: c.dim,
	};
	const mono = { fontFamily: 'JetBrains Mono' };
	const left = 470;
	const term = W - 2 * PAD - left - 48;
	const row = (line: Line) =>
		'cmd' in line
			? el('div', { display: 'flex', color: t.text }, [el('span', { color: t.muted, marginRight: 10 }, '$'), el('span', { width: term - 70 }, line.cmd)])
			: el(
					'div',
					{ display: 'flex', width: term - 50, color: line.tone === 'ok' ? t.ok : line.tone === 'bad' ? t.bad : line.tone === 'dim' ? t.muted : t.text },
					line.out,
				);
	const tree = el('div', { width: W, height: H, display: 'flex', background: l.page, fontFamily: 'Inter' }, [
		el('div', { width: W - 2 * PAD, height: H - 2 * PAD, margin: PAD, display: 'flex', alignItems: 'center' }, [
			el('div', { width: left, height: H - 2 * PAD, display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }, [
				el('div', { display: 'flex', alignItems: 'center', fontSize: 22, ...mono, fontWeight: 600, letterSpacing: 1.5 }, [
					el('img', { width: 56, height: 56, marginRight: 18 }, undefined, { src: mark, width: 56, height: 56 }),
					el('span', { color: l.accent }, 'SPINBOX'),
				]),
				el('div', { display: 'flex', flexDirection: 'column', ...mono, fontWeight: 600, fontSize: 52, lineHeight: 1.12, letterSpacing: -1 }, [
					el('div', { color: l.text }, lead),
					el('div', { color: l.accent, marginTop: 6 }, rest),
				]),
				el('div', { fontSize: 18, color: l.muted, ...mono, fontWeight: 600 }, site),
			]),
			el(
				'div',
				{ width: term, marginLeft: 48, display: 'flex', flexDirection: 'column', background: t.page, borderRadius: 10, border: `1px solid ${t.edge}` },
				[
					el(
						'div',
						{ display: 'flex', alignItems: 'center', height: 40, paddingLeft: 18, borderBottom: `1px solid ${t.edge}`, color: t.muted, fontSize: 15, ...mono },
						[
							el('div', { width: 10, height: 10, borderRadius: 5, background: t.edge, marginRight: 7 }),
							el('div', { width: 10, height: 10, borderRadius: 5, background: t.edge, marginRight: 7 }),
							el('div', { width: 10, height: 10, borderRadius: 5, background: t.edge, marginRight: 16 }),
							'api — spin',
						],
					),
					el(
						'div',
						{ display: 'flex', flexDirection: 'column', padding: '18px 22px 22px', fontSize: 15, lineHeight: 1.6, ...mono, fontWeight: 400 },
						session.map(row),
					),
				],
			),
		]),
	]);
	const svg = await satori(tree as never, { width: W, height: H, fonts });
	return sharp(Buffer.from(svg)).png().toBuffer();
}
