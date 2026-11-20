import type { APIRoute } from 'astro';
import { landingCard } from '../lib/og';

// The landing page's social card: the hero's still, in light (src/lib/og.ts). Head.astro points the
// landing page's og:image here; every other page's is in cards/.
export const GET: APIRoute = async () => new Response(new Uint8Array(await landingCard()), { headers: { 'Content-Type': 'image/png' } });
