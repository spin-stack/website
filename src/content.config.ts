import { defineCollection } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';
import { z } from 'astro/zod';

// Every page carries a description: it is the page's line in llms.txt and its meta
// description, and a page without one fails the build rather than leaving a blank.
export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		schema: docsSchema({ extend: z.object({ description: z.string().min(20) }) }),
	}),
};
