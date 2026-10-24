import constants from '../data/constants.json';

// The figures a page states that the code decides, as tools/refgen read them from spin and
// spin-terraform-aws: a page shows one with <Value name="..." /> instead of typing it, so the
// page moves when the code does.

/** The figure name, as a page writes it; a name refgen does not write fails the build. */
export function value(name: string): string {
	const c = (constants as Record<string, { text: string } | undefined>)[name];
	if (!c) throw new Error(`<Value name="${name}"> is no figure in src/data/constants.json`);
	return c.text;
}
