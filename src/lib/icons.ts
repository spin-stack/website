// The landing page's few icons, drawn here (24-unit strokes in the current colour) so the site
// loads nothing from anyone else. Icon.astro draws one; Answers.astro puts one on each answer.
export const icons = {
	machine: 'M4 5h16v10H4zM8 19h8M12 15v4M7 9l2 2-2 2M11 13h4',
	memory: 'M6 4h12v16H6zM9 8h6M9 12h6M9 16h3M3 8h3M3 12h3M3 16h3M18 8h3M18 12h3M18 16h3',
	diff: 'M7 4v10M7 18v2M4 7h6M17 4v16M14 12h6M12 2v20',
	shield: 'M12 3l7 3v6c0 4.5-3 7.5-7 9-4-1.5-7-4.5-7-9V6zM9 12l2 2 4-4',
	key: 'M14 10a4 4 0 1 0-1.2 2.8L20 20M17 17l2-2M15 15l1.5-1.5',
	agent: 'M12 3v3M7 7h10a3 3 0 0 1 3 3v6a3 3 0 0 1-3 3H7a3 3 0 0 1-3-3v-6a3 3 0 0 1 3-3zM9 12h.01M15 12h.01M9 16h6',
	cloud: 'M7 18a5 5 0 0 1-.6-10A6 6 0 0 1 18 9a4.5 4.5 0 0 1-.5 9zM12 11v5M10 14l2 2 2-2',
	move: 'M4 12h16M16 8l4 4-4 4M8 8l-4 4 4 4',
	moon: 'M20 14.5A8 8 0 1 1 9.5 4a6.5 6.5 0 0 0 10.5 10.5z',
	chip: 'M7 7h10v10H7zM10 10h4v4h-4zM9 3v4M15 3v4M9 17v4M15 17v4M3 9h4M3 15h4M17 9h4M17 15h4',
	kernel: 'M4 5h16v14H4zM7 10l3 2-3 2M12 15h5',
	nogpu: 'M3 8h18v9H3zM7 11h3v3H7zM13 11h4M13 14h4M4 3l16 18',
	region: 'M12 21s-7-6.5-7-12a7 7 0 0 1 14 0c0 5.5-7 12-7 12zM12 6.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5z',
} as const;

export type IconName = keyof typeof icons;

export function iconSVG(name: IconName): string {
	return `<svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="${icons[name]}"></path></svg>`;
}
