import reference from '../data/reference.json';

// What the site documents, as tools/refgen wrote it: one sentence, the same on the page and in
// its Markdown twin.
const { counts, spin } = reference;

/** The feature board's counts and the release, with the status page as Markdown links. */
export const statusLine =
	`**${counts.built} features built**, each with a test or a command that proves it` +
	(counts.partial ? `, ${counts.partial} partly built` : '') +
	`, ${counts.notBuilt} not built yet and ${counts.limits} limits, on the [status page](/reference/status/). ` +
	`An installation runs release \`${spin.release}\`; the reference is generated from spin \`${spin.commit.slice(0, 9)}\`.`;
