import { readFileSync } from 'node:fs';
import { experimental_AstroContainer as AstroContainer } from 'astro/container';
import satori from 'satori';
import sharp from 'sharp';
import Room from '../components/Room.astro';
import { layouts, STILL, tilted } from './film';

// The landing page's social card: the hero's still - the terminal open, every actor wired - in the
// light theme, drawn from the same component the page serves. The page's colours are CSS variables
// and colour mixes, which no rasterizer here reads, so the card gives each class its light colour
// outright; and no font is installed where the site is built, so each label is drawn as its
// glyphs' paths by satori, as the other cards are (card.ts).

const W = 1200;
const H = 630;

// The light theme's tokens, as task reference copies them from brand.
const tokens = readFileSync('src/styles/brand-tokens.css', 'utf8');
const lightBlock = tokens.slice(tokens.indexOf(":root[data-theme='light']"));
function token(name: string, from = lightBlock): string {
	const hex = from.match(new RegExp(`--spin-${name}:\\s*(#[0-9A-Fa-f]{6})`))?.[1] ?? tokens.match(new RegExp(`--spin-${name}:\\s*(#[0-9A-Fa-f]{6})`))?.[1];
	if (!hex) throw new Error(`src/styles/brand-tokens.css has no --spin-${name}`);
	return hex;
}
const c = {
	page: token('page'),
	surface: token('surface'),
	raised: token('raised'),
	border: token('border'),
	strong: token('border-strong'),
	text: token('text'),
	muted: token('muted'),
	label: token('label'),
	accent: token('accent'),
	onAccent: token('on-accent'),
	success: token('success'),
	warning: token('warning'),
	error: token('error'),
	blue: token('brand-blue'),
	cyan: token('brand-cyan'),
	deep: token('brand-deep'),
	frost: token('brand-frost'),
	termBg: token('terminal-background'),
	termFg: token('terminal-foreground'),
};

/** The component's classes, in light, with a stroke's width in the card's units (u per pixel). */
const sheet = (u: number) => `
.frame{fill:${c.surface};stroke:${c.border};stroke-width:${u}}
.card{fill:${c.raised};stroke:${c.border};stroke-width:${u}}
.sheet{fill:${c.raised};stroke:${c.strong};stroke-width:${u}}
.input{fill:${c.page};stroke:${c.strong};stroke-width:${u}}
.bar,.gauge{fill:${c.raised}}
.dot{fill:${c.border}}
.gauge-on{fill:${c.blue}}
.tick,.gate{stroke:${c.text};stroke-width:${1.5 * u}}
.gate{fill:${c.text}}
.primary{fill:${c.accent}}
.secondary,.chip{fill:${c.page};stroke:${c.strong};stroke-width:${u}}
.term{fill:${c.termBg}}
.hair{fill:none;stroke:${c.border};stroke-width:${u}}
.wire,.outline,.ref,.arrowhead{fill:none;stroke:${c.strong};stroke-width:${1.5 * u}}
.outline{stroke:${c.label}}
.ref{stroke:${c.cyan};stroke-width:${u}}
.pulse,.floor,.flow{display:none}
.pill{fill:${c.raised};stroke:${c.strong};stroke-width:${u}}
.side-deep{fill:${c.deep}}
.side-blue,.top-blue{fill:${c.blue}}
.top-cyan{fill:${c.cyan}}
.layer path,.layer ellipse{fill:${c.page};stroke:${c.label};stroke-width:${1.5 * u};stroke-dasharray:${6 * u} ${5 * u}}
.lock{fill:none;stroke:${c.text};stroke-width:${1.5 * u}}
.on-disk{stroke:${c.frost}}
.kchip rect{fill:${c.page};stroke-width:${1.5 * u}}
.k-A rect{stroke:${c.success}}
.k-M rect{stroke:${c.warning}}
.k-D rect{stroke:${c.error}}
.cursor{fill:${c.text};stroke:${c.page};stroke-width:${u}}
[stroke-dashoffset="1"],[stroke-dashoffset="-1"]{display:none}
`;
// A line not drawn yet is hidden outright: the card's rasterizer is not held to read pathLength.
// The one live thing at the still: the session's line, both halves of it.
const live = ['lBR', 'lRR', 'aBR', 'aRR'];

const face = (pkg: string, weight: 400 | 600) => readFileSync(`node_modules/@fontsource/${pkg}/files/${pkg}-latin-${weight}-normal.woff`);
const fonts = [
	{ name: 'Plex Sans', data: face('ibm-plex-sans', 400), weight: 400 as const },
	{ name: 'Plex Sans', data: face('ibm-plex-sans', 600), weight: 600 as const },
	{ name: 'Plex Mono', data: face('ibm-plex-mono', 400), weight: 400 as const },
	{ name: 'Plex Mono', data: face('ibm-plex-mono', 600), weight: 600 as const },
];

const entities: Record<string, string> = { '&amp;': '&', '&lt;': '<', '&gt;': '>', '&quot;': '"', '&#39;': "'", '&#x27;': "'" };
const decode = (s: string) => s.replace(/&(amp|lt|gt|quot|#39|#x27);/g, (e) => entities[e]);
const attr = (tag: string, name: string) => tag.match(new RegExp(`\\s${name}="([^"]*)"`))?.[1];

/** How a label is drawn, from its own classes and those of what holds it, as the page's CSS has it. */
function look(own: string[], held: string[]) {
	const has = (k: string) => own.includes(k);
	const under = (k: string) => held.includes(k);
	const kind = held.find((k) => /^k-[AMD]$/.test(k));
	const mono = has('mono') || under('mono') || under('tty') || !!kind;
	const size = under('tty') ? 14 : has('big') ? 22 : 16;
	const weight = has('h') || has('on-primary') || kind ? 600 : 400;
	const colour = has('on-primary')
		? c.onAccent
		: under('tty')
			? c.termFg
			: kind
				? { 'k-A': c.success, 'k-M': c.warning, 'k-D': c.error }[kind]
				: has('st-wait')
					? c.warning
					: has('st-run')
						? c.success
						: has('muted')
							? c.muted
							: has('label')
								? c.label
								: c.text;
	const middle = has('mid') || has('on-primary') || !!kind || under('mid');
	return { family: mono ? 'Plex Mono' : 'Plex Sans', size, weight, colour, middle };
}

/** A label as its glyphs: satori lays it out on a line 1.3 em high, its baseline 1.025 em down. */
async function glyphs(text: string, l: ReturnType<typeof look>): Promise<{ body: string; box: number }> {
	const box = 1000;
	const h = Math.ceil(l.size * 1.3);
	const out = await satori(
		{
			type: 'div',
			props: {
				style: {
					display: 'flex',
					width: box,
					height: h,
					justifyContent: l.middle ? 'center' : 'flex-start',
					fontFamily: l.family,
					fontSize: l.size,
					fontWeight: l.weight,
					lineHeight: 1.3,
					color: l.colour,
					whiteSpace: 'pre',
				},
				children: text,
			},
		} as never,
		{ width: box, height: h, fonts },
	);
	return { body: out.slice(out.indexOf('>') + 1, out.lastIndexOf('</svg>')), box };
}

/** Every <text> of the drawing, replaced by its glyphs where it stood. */
async function outline(svg: string): Promise<string> {
	const held: string[][] = [];
	let out = '';
	let at = 0;
	const tag = /<(\/?)([a-zA-Z]+)([^>]*?)(\/?)>/g;
	for (let m = tag.exec(svg); m; m = tag.exec(svg)) {
		const [whole, close, name, attrs, selfClosing] = m;
		if (close) {
			held.pop();
			continue;
		}
		const classes = (attr(attrs, 'class') ?? '').split(/\s+/);
		if (name !== 'text') {
			if (!selfClosing) held.push(classes);
			continue;
		}
		const end = svg.indexOf('</text>', m.index);
		const text = decode(svg.slice(m.index + whole.length, end).replace(/<[^>]*>/g, ''));
		out += svg.slice(at, m.index);
		at = end + '</text>'.length;
		tag.lastIndex = at;
		if (!text) continue;
		const l = look(classes, held.flat());
		const { body, box } = await glyphs(text, l);
		const x = Number(attr(attrs, 'x') ?? 0) - (l.middle ? box / 2 : 0);
		const y = Number(attr(attrs, 'y') ?? 0) - 1.025 * l.size;
		const own = attr(attrs, 'transform') ?? '';
		out += `<g transform="${own} translate(${x} ${y})">${body}</g>`;
	}
	return out + svg.slice(at);
}

export async function landingCard(): Promise<Buffer> {
	const container = await AstroContainer.create();
	const html = await container.renderToString(Room, { props: { t: STILL, px: 1600 } });
	let svg = html.slice(html.indexOf('<svg'), html.indexOf('</svg>') + '</svg>'.length);
	// The whole room, framed for the card: as wide as it needs, centred.
	const [rx, ry, rw, rh] = tilted(layouts.wide.room);
	const pad = 24;
	const vw = Math.max(rw + 2 * pad, ((rh + 2 * pad) * W) / H);
	const vh = (vw * H) / W;
	const box = [rx + rw / 2 - vw / 2, ry + rh / 2 - vh / 2, vw, vh];
	const u = vw / W;
	svg = svg
		.replace(/<title[\s\S]*?<\/desc>/, '')
		.replace(
			/<svg[^>]*>/,
			`<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="${box.join(' ')}"><style type="text/css">${sheet(u)}${live.map((id) => `[data-f="${id}"]`).join(',')}{stroke:${c.accent}}</style><rect x="${box[0]}" y="${box[1]}" width="${vw}" height="${vh}" fill="${c.page}"/>`,
		);
	return sharp(Buffer.from(await outline(svg)))
		.png()
		.toBuffer();
}
