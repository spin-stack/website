// The cost page's scenarios, from src/data/pricing.json (tools/pricing): the machines the module
// starts by default, priced on demand in each region Spin OS has images in, as the lists the AWS
// pricing page loads had them on the record's date. Three months, each the one before plus
// something: nothing running, one runner on working hours, the most runners around the clock.
// Scenarios.astro draws them; the page's Markdown twin is scenariosMarkdown.
import pricing from '../data/pricing.json';

/** A month as AWS bills it. */
const hours = 730;
/** Working hours in a month: eight a day, twenty-two days. */
const workingHours = 8 * 22;
// What the installation's smaller parts are billed at, from AWS's pricing pages as of the
// record's date: the public IPv4 address the proxy's elastic IP holds (an hour), and a Route 53
// hosted zone (a month, and the installation has two).
const ipv4Hourly = 0.005;
const zoneMonthly = 0.5;
const zones = 2;

export interface Scenario {
	name: string;
	what: string;
	/** Dollars a month, by region code. */
	monthly: Record<string, number>;
}

export const regions = Object.keys(pricing.regions).sort();
export const generated = pricing.generated;
export const published = pricing.published.slice(0, 10);
export const machines = pricing.machines;

function price(region: string, type: string): number {
	const p = (pricing.regions as Record<string, Record<string, { hourly: number }>>)[region]?.[type];
	if (!p) throw new Error(`pricing.json: no price for ${type} in ${region}`);
	return p.hourly;
}

/** The three scenarios, each a month in dollars per region. */
export function scenarios(): Scenario[] {
	const { controlPlane, proxy, runners, maxRunners } = machines;
	const runner = runners[0];
	const perRegion = (f: (region: string) => number) => Object.fromEntries(regions.map((r) => [r, f(r)]));
	const idle = (r: string) => (price(r, controlPlane) + price(r, proxy) + ipv4Hourly) * hours + zones * zoneMonthly;
	const n = maxRunners;
	return [
		{
			name: 'Nothing running',
			what: `The control plane (${controlPlane}) and the proxy (${proxy}) all month, the proxy's public IPv4 address, and two Route 53 zones. No runner is up.`,
			monthly: perRegion(idle),
		},
		{
			name: 'One runner, working hours',
			what: `The above, and one ${runner} up ${workingHours} hours: eight a day, twenty-two days, on demand.`,
			monthly: perRegion((r) => idle(r) + price(r, runner) * workingHours),
		},
		{
			name: n === 1 ? 'One runner around the clock' : `${n} runners around the clock`,
			what: `The above with ${n === 1 ? 'the runner' : `${n} runners (max_runners' default)`} up all month, on demand: the most the default installation can spend on runners.`,
			monthly: perRegion((r) => idle(r) + n * price(r, runner) * hours),
		},
	];
}

/** A dollar figure as the table shows it. */
export function dollars(n: number): string {
	return `$${n.toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`;
}

/** The scenarios as a Markdown table, for the page's twin. */
export function scenariosMarkdown(): string {
	const head = `| Scenario | What runs | ${regions.map((r) => `${r} ($/month)`).join(' | ')} |`;
	const rule = `|---|---|${regions.map(() => '---').join('|')}|`;
	const rows = scenarios().map((s) => `| **${s.name}** | ${s.what} | ${regions.map((r) => dollars(s.monthly[r])).join(' | ')} |`);
	return [head, rule, ...rows].join('\n');
}
