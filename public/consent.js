// Google Tag Manager, only after a reader says yes. Head.astro loads this file in production builds
// with the container's ID (data-id). Nothing from Google loads, and no cookie is set, until the
// banner's Accept (Footer.astro): then Consent Mode is told analytics may store, and gtm.js loads. The answer is
// kept in localStorage; "Cookie preferences" in the footer asks again, and a No after a Yes takes the
// consent back and clears Google's cookies. A file, not an inline script, so the
// Content-Security-Policy names it with 'self'.
(() => {
	const id = document.currentScript?.dataset.id;
	if (!id) return;
	const KEY = 'consent';
	window.dataLayer = window.dataLayer || [];
	function gtag() {
		// biome-ignore lint/complexity/noArguments: gtag reads an Arguments object from the layer, not an array.
		window.dataLayer.push(arguments);
	}
	gtag('consent', 'default', { ad_storage: 'denied', ad_user_data: 'denied', ad_personalization: 'denied', analytics_storage: 'denied' });

	let loaded = false;
	const load = () => {
		gtag('consent', 'update', { analytics_storage: 'granted' });
		if (loaded) return;
		loaded = true;
		window.dataLayer.push({ 'gtm.start': Date.now(), event: 'gtm.js' });
		const tag = document.createElement('script');
		tag.async = true;
		tag.src = `https://www.googletagmanager.com/gtm.js?id=${encodeURIComponent(id)}`;
		document.head.append(tag);
	};
	const forget = async () => {
		gtag('consent', 'update', { analytics_storage: 'denied' });
		if (!window.cookieStore) return;
		for (const { name, domain } of await cookieStore.getAll()) {
			if (name.startsWith('_ga')) await cookieStore.delete({ name, domain: domain ?? undefined, path: '/' });
		}
	};

	let answer = null;
	try {
		answer = localStorage.getItem(KEY);
	} catch {}
	if (answer === 'granted') load();

	addEventListener('DOMContentLoaded', () => {
		const banner = document.getElementById('consent');
		if (!banner) return;
		// Out of the footer, whose column is a stacking context the sidebar covers, to the page itself.
		document.body.append(banner);
		if (answer !== 'granted' && answer !== 'denied') banner.hidden = false;
		banner.addEventListener('click', (e) => {
			const choice = e.target.closest('button')?.dataset.consent;
			if (!choice) return;
			try {
				localStorage.setItem(KEY, choice);
			} catch {}
			banner.hidden = true;
			if (choice === 'granted') load();
			else if (answer === 'granted' || loaded) forget();
			answer = choice;
		});
		for (const open of document.querySelectorAll('[data-consent-open]')) {
			open.hidden = false;
			open.addEventListener('click', (e) => {
				e.preventDefault();
				banner.hidden = false;
				banner.querySelector('button')?.focus();
			});
		}
	});
})();
