import { defineCollection } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';
import { z } from 'astro/zod';

// Every page carries a description: it is the page's line in llms.txt and its meta
// description, and a page without one fails the build rather than leaving a blank. Under 70
// characters a search result fills the rest with whatever it picks from the page; over 155 it
// is cut. metaTitle is the page's <title> where its title alone says too little ("Cost",
// "spin cp"); Head.astro holds what a search result shows to 30 to 65 characters.
export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		// A generated page names its source as repository:path (tools/refgen), which the footer links.
		// A hand-written one names what it reads, each with the release it was last read against,
		// which refgen holds it to (tools/refgen/reads.go).
		schema: docsSchema({
			extend: z.object({
				description: z.string().min(70).max(155),
				metaTitle: z.string().optional(),
				source: z
					.string()
					.regex(/^[\w-]+:\S+$/)
					.optional(),
				reads: z.array(z.string().regex(/^[\w-]+:[^@\s]+@[\w.-]+$/)).optional(),
			}),
		}),
	}),
};
