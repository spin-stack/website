import reference from '../data/reference.json';

// What the site documents, as tools/refgen wrote it: one sentence, the same on the page and in
// its Markdown twin.
const { counts, spin } = reference;

/**
 * What the reference is generated from: a release, or a commit of spin main that no release has
 * yet, which is not what an installation runs.
 */
export const generatedFrom = spin.tag
	? `spin \`${spin.tag}\` (\`${spin.commit.slice(0, 9)}\`)`
	: `spin main at \`${spin.commit.slice(0, 9)}\`, not released`;

/** The feature board's counts and the release, with the status page as Markdown links. */
export const statusLine =
	`**${counts.built} features built**, each with a test or a command that proves it, ${counts.endToEnd} of them with an end-to-end test` +
	(counts.partial ? `, ${counts.partial} partly built` : '') +
	`, ${counts.notBuilt} not built yet and ${counts.limits} limits, on the [status page](/reference/status/). ` +
	`An installation runs release \`${spin.installs}\`; the reference is generated from ${generatedFrom}.`;
