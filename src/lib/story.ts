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

// What the workspace's root writes in the story: one file added, one modified, one removed, in
// the order of the diff golden's lines, which is the order the hero lands them in the top layer.
export const writes = ['echo hello > /etc/greeting', 'echo "# edited" >> /etc/hosts', 'rm /etc/issue'];

export const steps: Step[] = [
	{
		label: 'Snapshot',
		title: 'Take a snapshot, memory and all',
		claim: 'A machine of its own, and a snapshot of its disk with the memory of every running process.',
		feature: 'Snapshots with memory, resumed with their processes',
		commands: ['spin workspace create --name api --wait', 'spin snapshot create api --description clean --with-memory --wait'],
	},
	{
		label: 'Break it',
		title: 'Let anyone break anything',
		claim: "A person or an agent is root in a KVM machine with its own kernel, not in a container that shares the host's.",
		feature: 'A machine per workspace: KVM, its own kernel, root, a persistent disk, SSH',
		commands: [`spin exec api -- sh -c '${writes.join('; ')}'`, 'spin snapshot create api --description after --wait'],
	},
	{
		label: 'Diff',
		title: 'See what changed',
		claim: 'The files added, modified or removed between two snapshots, read by a separate machine.',
		feature: 'A diff of the files that changed between two snapshots',
		commands: ['spin snapshot diff api <after-id> --from <clean-id> --under /etc'],
		output: 'diff.txt',
	},
	{
		label: 'Roll back',
		title: 'Go back as if it never happened',
		claim: 'The disk as it was, and the processes running where they were.',
		feature: 'Restore a workspace to a snapshot',
		commands: ['spin workspace restore api <clean-id> --with-memory', 'spin workspace delete api'],
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

/** A chapter of the hero's film: its button, when it starts in the loop, the CLI command that does
 * what the browser does in it (none where nobody asks for anything), and one sentence a reader
 * can check against the concept pages. */
export interface Chapter {
	name: string;
	at: number;
	cmd?: string;
	say: string;
}

export const chapters: Chapter[] = [
	{
		name: 'Ask',
		at: 0,
		cmd: steps[0].commands[0],
		say: 'Create workspace asks the control plane for one. It exists, and has not run yet.',
	},
	{ name: 'Place', at: 3, say: 'The control plane places it on an active runner with room for its disk and memory.' },
	{
		name: 'Disk',
		at: 7,
		say: "Its disk is a clone of the image, which copies nothing: it reads through the image's layer and writes only its own.",
	},
	{ name: 'Machine', at: 11, say: 'The runner starts it as a QEMU/KVM machine, confined to its own user, cgroup and empty network namespace.' },
	{
		name: 'Boot',
		at: 14,
		say: "Its kernel hands over to vminitd, and vminitd to the image's systemd. The supervisor talks to the runner over vsock; nftables, outside the machine, allows only what a policy names.",
	},
	{
		name: 'Open',
		at: 17,
		cmd: 'spin ssh api',
		say: "Your passkey adds this browser's key for twelve hours. The browser and the runner each dial out to the relay: nothing dials in.",
	},
	{ name: 'Write', at: 21, cmd: steps[1].commands[0], say: 'What you do as root lands in the top layer: a file added, one modified, one removed.' },
	{
		name: 'Keep',
		at: 24,
		cmd: 'spin snapshot create api --wait',
		say: 'A snapshot seals the top layer and uploads it to the bucket, beside the image. The machine writes on a new one.',
	},
];
