// The Content-Security-Policy's two halves that are not a list of addresses. Astro hashes the
// scripts it bundles, but not one written is:inline, and Starlight writes five (the theme before
// the first paint, the search shortcut, the sidebar's scroll), so their hashes are read from
// Starlight's own components, whatever version is installed. cspCheck fails the build on an
// inline script in a page that its policy does not allow, which a browser would refuse to run.
import { createHash } from 'node:crypto';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const sha256 = (text) => `sha256-${createHash('sha256').update(text).digest('base64')}`;

/** The hash of every is:inline script in Starlight's components. */
export function starlightInlineHashes() {
	const dist = fileURLToPath(new URL('..', import.meta.resolve('@astrojs/starlight/components/Head.astro')));
	const hashes = new Set();
	for (const file of readdirSync(dist, { recursive: true })) {
		if (!String(file).endsWith('.astro')) continue;
		for (const m of readFileSync(join(dist, String(file)), 'utf8').matchAll(/<script is:inline[^>]*>([\s\S]*?)<\/script>/g)) {
			hashes.add(sha256(m[1]));
		}
	}
	return [...hashes];
}

/** An integration that fails the build on an inline script a page's policy does not allow. */
export function cspCheck() {
	return {
		name: 'csp-check',
		hooks: {
			'astro:build:done': ({ dir, logger }) => {
				const root = fileURLToPath(dir);
				const bad = [];
				for (const file of readdirSync(root, { recursive: true })) {
					if (!String(file).endsWith('.html')) continue;
					const html = readFileSync(join(root, String(file)), 'utf8');
					const policy = html.match(/<meta http-equiv="content-security-policy" content="([^"]*)"/i)?.[1];
					if (!policy) {
						bad.push(`${file}: no Content-Security-Policy`);
						continue;
					}
					for (const m of html.matchAll(/<script(?![^>]*\bsrc=)(?![^>]*type="application\/ld\+json")[^>]*>([\s\S]*?)<\/script>/g)) {
						if (!policy.includes(`'${sha256(m[1])}'`)) bad.push(`${file}: an inline script its policy does not allow: ${m[1].trim().slice(0, 60)}`);
					}
				}
				if (bad.length) throw new Error(`Content-Security-Policy:\n${bad.join('\n')}`);
				logger.info("every inline script is allowed by its page's policy");
			},
		},
	};
}
