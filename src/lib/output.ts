// What spin's CLI prints, copied by tools/refgen from the goldens spin's tests hold it to.
const files = import.meta.glob<string>('../data/output/*', { query: '?raw', import: 'default', eager: true });

/** The output called name, as the golden has it; a name no golden has fails the build. */
export function output(name: string): string {
	const text = files[`../data/output/${name}`];
	if (text === undefined) throw new Error(`no output ${name} in src/data/output`);
	return text.replace(/\n$/, '');
}
