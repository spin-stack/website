// The landing page's film: one workspace born, from a click in the browser to a process writing on
// its disk and a snapshot of it in the bucket. It plays once. Every attribute of the drawing is a pure function of the time
// (frame), so a scrub, a chapter button and the still the page is served with are the same call.
// A value that changes more than once is a sum of closed-form springs, one per change; a line is
// drawn by its dash offset over pathLength 1; the camera is the SVG's viewBox.
//
// The drawing's coordinates: each actor is drawn in its own (local) frame, and a layout puts the
// actors in the world - wide (16:10) side by side, tall (4:5) one under the other. Lines between
// actors and whatever flies along them are in the world frame.
import { writes } from './story';

/** The film's length in its own seconds: it ends on the whole room, the snapshot in the browser. */
export const END = 27.6;
/** The still the page is served with and stays at under reduced motion: the terminal open. */
export const STILL = 20.6;

type Pt = [number, number];
type Box = [number, number, number, number];
export type Attrs = Record<string, string | number>;
export type Frame = Record<string, Attrs>;

export type Actor = 'browser' | 'cp' | 'relay' | 'runA' | 'runB' | 'bucket';
type LineId = 'req' | 'rA' | 'rB' | 'bR' | 'rR' | 'up';

export interface Layout {
	name: 'wide' | 'tall';
	aspect: number;
	room: Box;
	at: Record<Actor, Box>;
	lines: Record<LineId, Pt[]>;
}

// Each actor's own drawing, in its local frame: shared with the component that draws them.
export const B = {
	w: 440,
	h: 380,
	top: 34,
	btn: [228, 236, 160, 38] as Box,
	open: [24, 92, 150, 38] as Box,
	snap: [184, 92, 160, 38] as Box,
	term: [24, 144, 392, 148] as Box,
	chip: [116, 312, 96, 26] as Box,
	sheet: [80, 34, 280, 150] as Box,
	cont: [260, 140, 84, 32] as Box,
	rest: [372, 330] as Pt,
};
export const CP = { w: 360, h: 220, row: [8, 80, 344, 40] as Box, chip: [172, 87, 104, 26] as Box };
export const R = {
	w: 460,
	h: 606,
	gauge: 208,
	user: [16, 76, 290, 330] as Box,
	cg: [28, 102, 266, 292] as Box,
	ns: [40, 128, 242, 254] as Box,
	egress: [322, 150, 122, 180] as Box,
	tok: [84, 16, 56, 28] as Box,
	// A layer: the top ellipse's centre, its radii and the height of its side.
	// The image's layer in the runner's cache, and the top of the workspace's own (its tip).
	cache: [70, 508] as Pt,
	vol: [220, 478] as Pt,
	lx: 56,
	ly: 14,
	side: 30,
	sup: [312, 500] as Pt,
	vs: [380, 500] as Pt,
};
export const BK = { h: 110, shelf: 86, top: 56, rx: 40, ry: 10, side: 16 };

const actorsW: Record<Actor, Box> = {
	browser: [10, 150, B.w, B.h],
	cp: [500, 150, CP.w, CP.h],
	relay: [600, 470, 160, 56],
	runA: [910, 30, R.w, 96],
	runB: [910, 140, R.w, R.h],
	bucket: [500, 766, 870, BK.h],
};
const actorsT: Record<Actor, Box> = {
	browser: [130, 40, B.w, B.h],
	cp: [40, 640, CP.w, CP.h],
	relay: [420, 700, 160, 56],
	runA: [70, 900, R.w, 96],
	runB: [70, 1030, R.w, R.h],
	bucket: [40, 1660, 520, BK.h],
};
const slot = (a: Record<Actor, Box>): Pt => [a.bucket[0] + a.bucket[2] - 140, a.bucket[1] + BK.top];
const sealedAt = (a: Record<Actor, Box>): Pt => [a.runB[0] + R.vol[0] + R.lx, a.runB[1] + R.vol[1] + R.ly];

// The wide room is 1430 wide with its margins: the whole of it at 16 units a label is 12 px on a
// hero 1078 px wide, the page's at 1440.
export const layouts: Record<'wide' | 'tall', Layout> = {
	wide: {
		name: 'wide',
		aspect: 1.6,
		room: [0, 30, 1390, 846],
		at: actorsW,
		lines: {
			req: [
				[450, 270],
				[500, 270],
			],
			rA: [
				[910, 78],
				[882, 78],
				[882, 210],
				[860, 210],
			],
			rB: [
				[910, 190],
				[894, 190],
				[894, 250],
				[860, 250],
			],
			bR: [
				[450, 498],
				[600, 498],
			],
			rR: [
				[910, 498],
				[760, 498],
			],
			up: [sealedAt(actorsW), [1382, sealedAt(actorsW)[1]], [1382, slot(actorsW)[1]], slot(actorsW)],
		},
	},
	tall: {
		name: 'tall',
		aspect: 0.8,
		room: [0, 20, 600, 1760],
		at: actorsT,
		lines: {
			req: [
				[230, 420],
				[230, 640],
			],
			rA: [
				[300, 900],
				[300, 860],
			],
			rB: [
				[70, 1070],
				[30, 1070],
				[30, 880],
				[100, 880],
				[100, 860],
			],
			bR: [
				[500, 420],
				[500, 700],
			],
			rR: [
				[530, 1100],
				[570, 1100],
				[570, 756],
			],
			up: [sealedAt(actorsT), [548, sealedAt(actorsT)[1]], [548, slot(actorsT)[1]], slot(actorsT)],
		},
	},
};

// --- time

const clamp = (x: number, a = 0, b = 1) => Math.min(b, Math.max(a, x));
const ease = (x: number) => {
	const c = clamp(x);
	return c < 0.5 ? 4 * c * c * c : 1 - (-2 * c + 2) ** 3 / 2;
};
/** A spring's step response from 0 to 1, critically damped unless z < 1. */
function S(tau: number, w = 14, z = 1): number {
	if (tau <= 0) return 0;
	if (z >= 1) return 1 - Math.exp(-w * tau) * (1 + w * tau);
	const wd = w * Math.sqrt(1 - z * z);
	return 1 - Math.exp(-z * w * tau) * (Math.cos(wd * tau) + ((z * w) / wd) * Math.sin(wd * tau));
}
/** A value that starts at v0 and is sent to each key's value at its time: one spring per change. */
function tr(t: number, v0: number, keys: [number, number, number?][], z = 1): number {
	let v = v0;
	let prev = v0;
	for (const [at, to, w] of keys) {
		v += (to - prev) * S(t - at, w ?? 14, z);
		prev = to;
	}
	return v;
}
/** A line's dash offset: 1 undrawn, 0 drawn from its start, -1 taken back towards its end. */
const drawn = (t: number, at: number, d = 0.5, back = Number.POSITIVE_INFINITY) => 1 - ease((t - at) / d) - ease((t - back) / 0.4);
/** A highlight's weight between two times: what the accent mixes into a line. */
const live = (t: number, from: number, to: number) => clamp(S(t - from, 12) - S(t - to, 12));
const lives = (t: number, spans: [number, number][]) => Math.max(0, ...spans.map(([a, b]) => live(t, a, b)));
/** The loop's last half second: a spring that has nearly come back is put exactly where it began. */
/** What rises out of a mask line: its offset below the line, 0 once it has risen. */
const rise = (t: number, at: number, h = 28, out = Number.POSITIVE_INFINITY) => h * (1 - S(t - at, 16)) - h * S(t - out, 16);

/** A word as it decodes: each letter a random glyph until its turn, from `at` on. */
const GLYPHS = '0123456789abcdef#$%&*+=<>';
function decoded(word: string, t: number, at: number): string {
	const step = Math.floor(t * 24);
	return [...word]
		.map((ch, j) => {
			if (t < at || t >= at + 0.15 + j * 0.05) return ch;
			const h = Math.imul(j + 1, 73856093) ^ Math.imul(step + 1, 19349663);
			return GLYPHS[(h >>> 0) % GLYPHS.length];
		})
		.join('');
}

// --- geometry

function lengths(p: Pt[]): number[] {
	const l = [0];
	for (let i = 1; i < p.length; i++) l.push(l[i - 1] + Math.hypot(p[i][0] - p[i - 1][0], p[i][1] - p[i - 1][1]));
	return l;
}
function along(p: Pt[], f: number): Pt {
	const l = lengths(p);
	const at = clamp(f) * l[l.length - 1];
	for (let i = 1; i < p.length; i++) {
		if (at <= l[i] || i === p.length - 1) {
			const s = l[i] === l[i - 1] ? 0 : (at - l[i - 1]) / (l[i] - l[i - 1]);
			return [p[i - 1][0] + (p[i][0] - p[i - 1][0]) * s, p[i - 1][1] + (p[i][1] - p[i - 1][1]) * s];
		}
	}
	return p[p.length - 1];
}
export const pathD = (p: Pt[]) => `M${p.map(([x, y]) => `${x} ${y}`).join('L')}`;
const union = (bs: Box[]): Box => {
	const x0 = Math.min(...bs.map((b) => b[0]));
	const y0 = Math.min(...bs.map((b) => b[1]));
	const x1 = Math.max(...bs.map((b) => b[0] + b[2]));
	const y1 = Math.max(...bs.map((b) => b[1] + b[3]));
	return [x0, y0, x1 - x0, y1 - y0];
};
const centre = (b: Box): Pt => [b[0] + b[2] / 2, b[1] + b[3] / 2];
const local = (a: Box, x: number, y: number): Pt => [a[0] + x, a[1] + y];
/** A path from a to b that goes along one axis and then the other. */
const elbow = (a: Pt, b: Pt, first: 'x' | 'y'): Pt[] => (first === 'x' ? [a, [b[0], a[1]], b] : [a, [a[0], b[1]], b]);

// --- the tilt

/** The room's tilt: turned by `turn` degrees, then pressed to `press` of its height. The room is
 * drawn flat and the tilt is one matrix over all of it; the camera frames the tilted room. */
const TILT = { turn: -10, press: 0.85 };
const rad = (TILT.turn * Math.PI) / 180;
const M = [Math.cos(rad), TILT.press * Math.sin(rad), -Math.sin(rad), TILT.press * Math.cos(rad)];
const tilt = ([x, y]: Pt): Pt => [M[0] * x + M[2] * y, M[1] * x + M[3] * y];
export function tilted(b: Box): Box {
	const ps = [tilt([b[0], b[1]]), tilt([b[0] + b[2], b[1]]), tilt([b[0], b[1] + b[3]]), tilt([b[0] + b[2], b[1] + b[3]])];
	return union(ps.map(([x, y]) => [x, y, 0, 0] as Box));
}

// --- the camera

// A whole shot frames the room whatever the hero's width, its labels a little under 12 px where the
// tilted room is wider than the camera may be.
type Shot = { at: number; see: (Actor | Box)[]; focus?: (Actor | Box)[]; w?: number; only?: 'wide' | 'tall'; whole?: boolean };

const rB = (L: Layout, y: number, h: number): Box => [L.at.runB[0], L.at.runB[1] + y, R.w, h];
function shots(L: Layout): Shot[] {
	const head = rB(L, 0, 80);
	const nvme = rB(L, 400, R.h - 400);
	const qemu = rB(L, 0, 410);
	const guest: Box = [L.at.runB[0] + R.user[0], L.at.runB[1] + R.user[1], R.egress[0] + R.egress[2] - R.user[0], R.user[3]];
	const start: Shot = { at: 0, see: ['browser'] };
	const all: Shot[] = [
		start,
		{ at: 1.0, see: ['browser', 'cp'], focus: ['cp'], w: 6 },
		{ at: 3.0, see: ['cp', 'runA', head], w: 5 },
		{ at: 6.8, see: [nvme, 'bucket'], focus: [nvme], w: 5 },
		{ at: 10.9, see: [qemu], w: 5 },
		{ at: 13.9, see: [guest], w: 5 },
		{ at: 16.4, see: ['browser', 'relay', 'runB'], focus: ['browser'], w: 5 },
		{ at: 18.4, see: ['relay'], focus: ['relay', head], w: 5, only: 'tall' },
		{ at: 19.4, see: ['browser'], w: 5, only: 'tall' },
		{ at: 22.6, see: [nvme], w: 5, only: 'tall' },
		{ at: 23.9, see: ['browser', 'cp', 'runB'], focus: ['browser'], w: 5 },
		{ at: 24.6, see: ['cp'], w: 6, only: 'tall' },
		{ at: 25.1, see: ['runB', 'bucket'], focus: [nvme], w: 5 },
		{ at: 26.0, see: ['bucket'], w: 6, only: 'tall' },
		{ at: 25.9, see: [L.room], focus: ['browser'], w: 5, whole: true },
	];
	return all.filter((s) => !s.only || s.only === L.name);
}
/** The widest the camera may be: the drawing's 16-unit labels never under 12 px. */
const widest = (px: number) => (px * 16) / 12;

function aim(L: Layout, s: Shot, px: number): [number, number, number] {
	const box = (a: Actor | Box): Box => tilted(typeof a === 'string' ? L.at[a] : a);
	const pad = L.name === 'wide' ? 20 : 10;
	const u = union(s.see.map(box));
	let w = Math.max(u[2] + 2 * pad, (u[3] + 2 * pad) * L.aspect);
	const most = s.whole ? Number.POSITIVE_INFINITY : Math.min(widest(px), tilted(L.room)[2]);
	let [cx, cy] = centre(u);
	w = Math.max(w, L.name === 'wide' ? 760 : 380);
	if (w > most) {
		// Narrower than the shot: keep its centre, moved only as far as keeps the focus in.
		w = most;
		const h = w / L.aspect;
		const fb = union((s.focus ?? s.see).map(box));
		cx = Math.min(Math.max(cx, fb[0] + fb[2] + pad - w / 2), fb[0] - pad + w / 2);
		cy = Math.min(Math.max(cy, fb[1] + fb[3] + pad - h / 2), fb[1] - pad + h / 2);
	}
	return [cx, cy, w];
}

// --- the frame

/** Everything the drawing is at time t, in seconds from 0 to END. */
export function frame(time: number, L: Layout, px: number): Frame {
	const t = clamp(time, 0, END);
	const r = t;
	const f: Frame = {};
	const A = L.at;
	const bw = (x: number, y: number) => local(A.browser, x, y);

	// The camera.
	const sh = shots(L).map((s) => ({ s, v: aim(L, s, px) }));
	const [cx, cy, cw] = [0, 1, 2].map((i) =>
		tr(
			t,
			sh[0].v[i],
			sh.slice(1).map(({ s, v }) => [s.at, v[i], s.w]),
		),
	);
	const ch = cw / L.aspect;
	const ink = cw / px; // world units per screen pixel: a stroke's width is in pixels times this
	f.svg = { viewBox: `${cx - cw / 2} ${cy - ch / 2} ${cw} ${ch}`, style: `--u:${ink}` };
	f.world = { transform: `matrix(${M.join(' ')} 0 0)` };

	for (const a of Object.keys(A) as Actor[]) f[`g-${a}`] = { transform: `translate(${A[a][0]} ${A[a][1]})` };
	f['g-row'] = f['g-cp'];
	f['g-runB2'] = f['g-runB'];
	const bk = A.bucket[2];
	f.bk = { width: bk };
	f.bkShelf = { x2: bk - 16 };
	f.bkSlot = { transform: `translate(${bk - 140} 0)` };

	// The lines that are always there: the browser's to the control plane, and the runners', which
	// they opened.
	f.lReq = { d: pathD(L.lines.req) };
	// What always travels them: the light that runs along an open line, each at its own phase.
	(
		[
			['fReq', L.lines.req, 0],
			['fRA', L.lines.rA, 0.45],
			['fRB', L.lines.rB, 0.9],
		] as [string, Pt[], number][]
	).forEach(([id, p, phase]) => {
		f[id] = { d: pathD(p), 'stroke-dashoffset': 0.12 - ((t * 0.55 + phase) % 1.6) };
	});
	const [fx, fy, fw, fh] = L.room;
	f.floor = { x: fx - 400, y: fy - 400, width: fw + 800, height: fh + 800 };
	f.lRA = { d: pathD(L.lines.rA), style: `--k:${lives(r, [[3.3, 4.1]])}` };
	f.lRB = { d: pathD(L.lines.rB), style: `--k:${lives(r, [[4.3, 6.6]])}` };
	f.fRA = { style: `--k:${lives(r, [[3.3, 4.1]])}` };
	f.fRB = { style: `--k:${lives(r, [[4.3, 6.6]])}` };

	// 1. Ask. The button squeezes into the request, which flies to the control plane and lands as
	// the workspace's row.
	const btn = B.btn;
	const btnC = bw(btn[0] + btn[2] / 2, btn[1] + btn[3] / 2);
	const req = L.lines.req;
	const p1path = [...elbow(btnC, req[0], L.name === 'wide' ? 'y' : 'x').slice(0, -1), ...req];
	const row: Box = [A.cp[0] + CP.row[0], A.cp[1] + CP.row[1], CP.row[2], CP.row[3]];
	const landed = req[req.length - 1];
	let c1: Pt = btnC;
	if (r >= 2.25) {
		const k = S(r - 2.25, 12);
		const rc = centre(row);
		c1 = [landed[0] + (rc[0] - landed[0]) * k, landed[1] + (rc[1] - landed[1]) * k];
	} else if (r >= 1.2) c1 = along(p1path, ease((r - 1.2) / 1.0));
	const w1 = tr(r, btn[2], [
		[0.62, 226],
		[2.25, row[2], 12],
	]);
	const h1 = tr(r, btn[3], [
		[0.62, 34],
		[2.25, row[3], 12],
	]);
	const rx1 = tr(r, 8, [
		[0.62, 17],
		[2.25, 6, 12],
	]);
	const box1 = { x: c1[0] - w1 / 2, y: c1[1] - h1 / 2, width: w1, height: h1, rx: rx1 };
	const vis1 = r >= 0.6 ? 'visible' : 'hidden';
	f.p1 = { ...box1, visibility: vis1, style: `--f:${1 - S(r - 0.62, 12)};--k:${lives(r, [[2.3, 3.3]])}` };
	f.p1c = box1;
	f.p1a = { x: c1[0], y: c1[1] + 6 + rise(r, -1, 34, 0.62), visibility: vis1 };
	f.p1b = { x: c1[0], y: c1[1] + 6 + rise(r, 0.72, 34, 2.28), visibility: vis1 };
	f.v1btn = { visibility: t >= 0.6 ? 'hidden' : 'visible' };
	f.rowIn = { transform: `translate(0 ${rise(r, 2.42, 34)})` };
	f.odo = {
		transform: `translate(0 ${
			-26 *
			tr(r, 0, [
				[7.1, 1],
				[16.6, 2],
			])
		})`,
	};

	// The browser's two pages: the create dialog, and the workspace's page it goes to.
	const slide = tr(t, 0, [[1.0, 1, 12]]);
	f.v1 = { transform: `translate(${-B.w * slide} 0)` };
	f.v2 = { transform: `translate(${B.w * (1 - slide)} 0)` };
	const grow = (at: number, b: Box, back = Number.POSITIVE_INFINITY) => {
		const s = S(r - at, 14) - S(r - back, 14);
		const [x, y] = centre(b);
		return { transform: `translate(${x} ${y}) scale(${s}) translate(${-x} ${-y})`, visibility: s > 0.001 ? 'visible' : 'hidden' };
	};
	f.v2open = grow(16.6, B.open);
	f.v2snap = grow(16.7, B.snap, 24.3);
	f.v2snap2 = grow(25.2, B.snap);

	// 2. Place: the control plane looks over its runners and sends the workspace to the one with
	// room, down the line that runner opened.
	const rBpath = [...L.lines.rB].reverse();
	const tokEnd = local(A.runB, R.tok[0] + R.tok[2] / 2, R.tok[1] + R.tok[3] / 2);
	let ct: Pt = rBpath[0];
	if (r >= 5.5) {
		const k = S(r - 5.5, 12);
		const e = rBpath[rBpath.length - 1];
		ct = [e[0] + (tokEnd[0] - e[0]) * k, e[1] + (tokEnd[1] - e[1]) * k];
	} else if (r >= 4.7) ct = along(rBpath, ease((r - 4.7) / 0.8));
	const ts = S(r - 4.6, 16);
	f.tok = { transform: `translate(${ct[0]} ${ct[1]}) scale(${ts})`, visibility: ts > 0.001 ? 'visible' : 'hidden' };
	f.bDisk = { width: R.gauge * tr(r, 0.3, [[9.2, 0.36]]) };
	f.bMem = { width: R.gauge * tr(r, 0.24, [[5.8, 0.44]]) };

	// 3. Disk: the clone reads through the image's layer the runner already has; the workspace's
	// own layer grows on top of it, and the two read-only disks arrive beside it.
	f.ref = { 'stroke-dashoffset': drawn(r, 8.0) };
	f.ghost = { 'stroke-dashoffset': drawn(r, 8.3) };
	const layer = (at: number, back = Number.POSITIVE_INFINITY) => {
		const s = S(r - at, 12) - S(r - back, 12);
		const base = R.vol[1] + R.side + R.ly;
		return { transform: `translate(0 ${base}) scale(1 ${s}) translate(0 ${-base})`, visibility: s > 0.001 ? 'visible' : 'hidden' };
	};
	f.tip = layer(9.0);
	f.tipLive = {
		style: `--k:${lives(r, [
			[9.0, 11.0],
			[21.0, 25.3],
		])};--s:${S(r - 25.35, 12)}`,
	};
	f.tipLock = grow(25.4, [R.vol[0] - 6, R.vol[1] + R.ly + 4, 12, 12]);
	f.tip2 = { ...layer(25.6), transform: `translate(0 ${-R.side}) ${layer(25.6).transform}` };
	f.tip2Live = { style: `--k:${live(r, 25.6, 27.0)}` };
	f.dSup = grow(10.0, [R.sup[0] - 26, R.sup[1] - 8, 52, 30]);
	f.dVs = grow(10.25, [R.vs[0] - 26, R.vs[1] - 8, 52, 30]);
	f.lbSup = { transform: `translate(0 ${rise(r, 10.1, 24)})` };
	f.lbVs = { transform: `translate(0 ${rise(r, 10.35, 24)})` };
	f.lbTip = { transform: `translate(0 ${rise(r, 9.2, 24)})` };

	// 4. Machine: the runner's box opens into the QEMU/KVM process and the three things it is
	// confined to, and the machine's drive is the layer on the NVMe.
	f.oUser = { 'stroke-dashoffset': drawn(r, 11.1), style: `--k:${live(r, 11.1, 13.9)}` };
	f.oCg = { 'stroke-dashoffset': drawn(r, 11.45) };
	f.oNs = { 'stroke-dashoffset': drawn(r, 11.8) };
	f.oUserT = { transform: `translate(0 ${rise(r, 11.3, 24)})` };
	f.oCgT = { transform: `translate(0 ${rise(r, 11.65, 24)})` };
	f.oNsT = { transform: `translate(0 ${rise(r, 12.0, 24)})` };
	f.drive = { 'stroke-dashoffset': drawn(r, 12.6) };

	// 5. Boot: kernel, vminitd, systemd, the supervisor and its vsock line out through the machine's
	// wall; the disks vminitd mounts read-only; the egress namespace's gate, closed.
	['kernel', 'vminitd', 'systemd', 'supervisor'].forEach((word, i) => {
		const at = [14.0, 14.5, 15.15, 15.6][i];
		f[`g${i}`] = { transform: `translate(0 ${rise(r, at, 26)})` };
		f[`gt${i}`] = { text: decoded(word, r, at) };
	});
	f.eSup = { 'stroke-dashoffset': drawn(r, 14.6, 0.4) };
	f.eVs = { 'stroke-dashoffset': drawn(r, 14.75, 0.4) };
	f.vs = { 'stroke-dashoffset': drawn(r, 15.7), style: `--k:${live(r, 15.7, 16.9)}` };
	f.vsT = grow(16.0, [-26, 126, 18, 48]);
	f.eg = grow(16.0, R.egress);
	f.veth = { 'stroke-dashoffset': drawn(r, 16.2, 0.4) };

	// 6. Open. Open terminal; the browser's passkey sheet; then the browser and the runner each
	// dial the relay, and the two lines are one.
	f.sheet = {
		transform: `translate(0 ${
			-(B.sheet[1] + B.sheet[3]) *
			(1 -
				tr(r, 0, [
					[17.55, 1],
					[18.3, 0],
				]))
		})`,
	};
	f.lBR = { d: pathD(L.lines.bR), 'stroke-dashoffset': drawn(t, 18.5, 0.6), style: `--k:${live(r, 19.3, 21.0)}` };
	f.lRR = { d: pathD(L.lines.rR), 'stroke-dashoffset': drawn(r, 18.75, 0.6), style: `--k:${live(r, 19.3, 21.0)}` };
	const arrow = (id: string, p: Pt[], at: number, back = Number.POSITIVE_INFINITY) => {
		const [a, b] = p.slice(-2);
		const deg = (Math.atan2(b[1] - a[1], b[0] - a[0]) * 180) / Math.PI;
		const s = S(r - at, 18) - S(r - back, 18);
		f[id] = { transform: `translate(${b[0]} ${b[1]}) rotate(${deg}) scale(${s})`, visibility: s > 0.001 ? 'visible' : 'hidden' };
	};
	arrow('aBR', L.lines.bR, 19.05);
	arrow('aRR', L.lines.rR, 19.3);
	const pane = B.term[3] * tr(r, 0, [[19.6, 1, 10]]);
	f.termBg = { height: pane };
	f.termClip = { height: pane };

	// 7. Write: each command typed in the terminal goes down the session to the machine and lands
	// in the top layer as what it did to the file.
	const prompt = 'root@api:~# ';
	const ends: number[] = [];
	writes.forEach((cmd, i) => {
		const start = 21.0 + i * 1.0;
		const end = start + cmd.length / 42;
		ends.push(end);
		const shown = r < start ? '' : cmd.slice(0, Math.round(clamp((r - start) / (end - start)) * cmd.length));
		const there = i === 0 ? r >= 20.0 : r >= ends[i - 1] + 0.05;
		f[`t${i}`] = { text: there ? prompt + shown : '' };
		const c = S(r - (end + 0.5), 16);
		const [x, y] = [R.vol[0] - 45 + i * 32 + 13, R.vol[1] + R.ly + 15];
		f[`c${i}`] = { transform: `translate(${x} ${y}) scale(${c})`, visibility: c > 0.001 ? 'visible' : 'hidden' };
	});
	f[`t${writes.length}`] = { text: r >= ends[ends.length - 1] + 0.05 ? prompt : '' };
	f.t0r = { transform: `translate(0 ${rise(r, 20.0, 24)})` };
	const pulse = (id: string, p: Pt[], from: number, to: number) => {
		let on = false;
		let off = 1;
		for (const e of ends) {
			const k = (r - (e + from)) / (to - from);
			if (k >= 0 && k <= 1) {
				on = true;
				off = 0.08 - 1.08 * ease(k);
			}
		}
		f[id] = { d: pathD(p), 'stroke-dashoffset': off, visibility: on ? 'visible' : 'hidden' };
	};
	pulse('pBR', L.lines.bR, 0, 0.25);
	pulse('pRR', [...L.lines.rR].reverse(), 0.25, 0.5);

	// 8. Keep: Create snapshot goes to the control plane and down the runner's line; the runner
	// seals the top layer, the machine writes on a new one, and the sealed layer goes up to the
	// bucket, beside the image. The snapshot appears in the browser, and the browser is back at the
	// create dialog.
	const snap = B.snap;
	const snapC = bw(snap[0] + snap[2] / 2, snap[1] + snap[3] / 2);
	const p2path = [...elbow(snapC, req[0], L.name === 'wide' ? 'y' : 'x').slice(0, -1), ...req, [landed[0], rBpath[0][1]] as Pt, ...rBpath];
	const c2 = r >= 24.5 ? along(p2path, ease((r - 24.5) / 0.85)) : snapC;
	const w2 = tr(r, snap[2], [
		[24.32, 222],
		[25.3, 0, 16],
	]);
	const h2 = tr(r, snap[3], [
		[24.32, 34],
		[25.3, 0, 16],
	]);
	const box2 = { x: c2[0] - w2 / 2, y: c2[1] - h2 / 2, width: w2, height: h2, rx: Math.min(17, h2 / 2) };
	const vis2 = r >= 24.3 && r < 25.6 ? 'visible' : 'hidden';
	f.p2 = { ...box2, visibility: vis2 };
	f.p2c = box2;
	f.p2a = { x: c2[0], y: c2[1] + 6 + rise(r, -1, 34, 24.32), visibility: vis2 };
	f.p2b = { x: c2[0], y: c2[1] + 6 + rise(r, 24.42, 34), visibility: vis2 };

	f.lUp = { d: pathD(L.lines.up), 'stroke-dashoffset': drawn(r, 25.7, 0.4) };
	const upOn = r >= 25.8 && r < 26.55;
	const cu = along(L.lines.up, ease((r - 25.8) / 0.7));
	f.up = { transform: `translate(${cu[0]} ${cu[1]})`, visibility: upOn ? 'visible' : 'hidden' };
	f.bSnap = { visibility: r >= 26.55 ? 'visible' : 'hidden' };
	f.bSnapT = { transform: `translate(0 ${rise(r, 26.6, 24)})` };
	const from = slot(A);
	const to = bw(B.chip[0] + B.chip[2] / 2, B.chip[1] + B.chip[3] / 2);
	const k = ease((r - 26.6) / 0.4);
	const sOn = r >= 26.6 && r < 27.0;
	f.sc = { transform: `translate(${from[0] + (to[0] - from[0]) * k} ${from[1] + (to[1] - from[1]) * k})`, visibility: sOn ? 'visible' : 'hidden' };
	f.v2chip = { visibility: r >= 27.0 ? 'visible' : 'hidden' };

	// The cursor: the person's hand, the same size on screen whatever the camera does, and not tilted
	// with the room.
	const cur: [number, Pt][] = [
		[0.15, [btnC[0] + 10, btnC[1] + 6]],
		[0.95, bw(B.rest[0] + 20, B.rest[1] + 10)],
		[16.9, bw(B.open[0] + 90, B.open[1] + 24)],
		[17.75, bw(B.cont[0] + 50, B.cont[1] + 20)],
		[18.35, bw(300, 250)],
		[23.95, bw(snap[0] + 90, snap[1] + 24)],
		[24.6, bw(300, 366)],
	];
	const rest = bw(...B.rest);
	const px0 = tr(
		t,
		rest[0],
		cur.map(([at, p]) => [at, p[0], 9]),
	);
	const py0 = tr(
		t,
		rest[1],
		cur.map(([at, p]) => [at, p[1], 9]),
	);
	const press = 1 - 0.18 * Math.max(...[0.6, 17.35, 18.15, 24.3].map((c) => live(t, c - 0.02, c + 0.1)));
	const [qx, qy] = tilt([px0, py0]);
	f.cur = { transform: `translate(${qx} ${qy}) scale(${ink * press})` };
	return f;
}
