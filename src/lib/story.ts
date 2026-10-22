// The landing page's story, in four steps: the commands a person types, what `spin snapshot diff`
// prints (the golden spin's tests hold, by way of src/data/output), and the claim each step
// makes, tied to the feature board's row that says it is built. Story.astro shows it; the page's
// Markdown twin is storyMarkdown.
import { featureHref } from './feature-links.mjs';
import { output } from './output';

export interface Step {
	label: string;
	title: string;
	claim: string;
	feature: string;
	commands: string[];
	output?: string;
}

export const steps: Step[] = [
	{
		label: 'Snapshot',
		title: 'Take a snapshot, memory and all',
		claim: 'A machine of its own, and a snapshot of its disk with the memory of every running process.',
		feature: 'Snapshots with memory, resumed with their processes',
		commands: [
			'spin workspace create --name demo --wait',
			'spin snapshot create demo --description clean --with-memory --wait',
		],
	},
	{
		label: 'Break it',
		title: 'Let anyone break anything',
		claim: 'A person or an agent is root in a KVM machine with its own kernel: nothing they do reaches another workspace.',
		feature: 'A machine per workspace: KVM, its own kernel, root, a persistent disk, SSH',
		commands: [
			"spin exec demo -- sh -c 'echo hello > /etc/greeting; rm /etc/issue'",
			'spin snapshot create demo --description after --wait',
		],
	},
	{
		label: 'Diff',
		title: 'See exactly what changed',
		claim: 'Every file added, modified or removed between two snapshots, read by a separate machine.',
		feature: 'A diff of the files that changed between two snapshots',
		commands: ['spin snapshot diff demo <after-id> --from <clean-id> --under /etc'],
		output: 'diff.txt',
	},
	{
		label: 'Roll back',
		title: 'Go back as if it never happened',
		claim: 'The disk as it was, and the processes running where they were.',
		feature: 'Restore a workspace to a snapshot',
		commands: ['spin workspace restore demo <clean-id> --with-memory', 'spin workspace delete demo'],
	},
];

/** A step's output as the golden has it, its lines split; none for a step that shows none. */
export function stepOutput(step: Step): string[] {
	return step.output ? output(step.output).split('\n') : [];
}

/** The status page's link for a step's claim. */
export function stepHref(step: Step): string {
	return featureHref(step.feature).href;
}

/** The story as Markdown: each step's claim with its link, then the commands and the output. */
export function storyMarkdown(): string {
	return steps
		.map((s, i) => {
			const out = stepOutput(s);
			return [
				`${i + 1}. **${s.title}.** ${s.claim} ([status](${stepHref(s)}))`,
				'',
				'```bash',
				...s.commands,
				'```',
				...(out.length ? ['', '```text', ...out, '```'] : []),
			].join('\n');
		})
		.join('\n\n');
}
