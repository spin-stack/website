import { defineCollection } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';
import { z } from 'astro/zod';

// Every page carries a description: it is the page's line in llms.txt and its meta
// description, and a page without one fails the build rather than leaving a blank.
export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		// A generated page names its source as repository:path (tools/refgen), which the footer links.
		// A hand-written one names what it reads, each with the release it was last read against,
		// which refgen holds it to (tools/refgen/reads.go).
		schema: docsSchema({
			extend: z.object({
				description: z.string().min(20),
				source: z.string().regex(/^[\w-]+:\S+$/).optional(),
				reads: z.array(z.string().regex(/^[\w-]+:[^@\s]+@[\w.-]+$/)).optional(),
			}),
		}),
	}),
};
