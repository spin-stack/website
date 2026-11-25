// The cookie banner and Google Tag Manager, the same file on spinbox.dev, docs. and blog.: one
// answer, in the cc_cookie cookie on spinbox.dev, holds for all three. The banner is
// vanilla-cookieconsent (cookieconsent.umd.js and cookieconsent.css beside this file: 3.3.0, MIT,
// copies from npm), served from this site so the Content-Security-Policy names it with 'self'.
// Its settings are the ones the GTM-N6G9KXJ7 container's own banner had, without the marketing
// category: nothing here shows ads. Tag Manager loads on every page with every kind of storage
// denied; accepting Analytics grants analytics_storage, and the container's tags wait for it.
// Rejecting it after accepting deletes Google's cookies (autoClear). "Cookie preferences" in the
// footer is an <a data-cc="show-preferencesModal">, which the library opens.
// Loaded deferred, after cookieconsent.umd.js, with the container's ID in data-id; off spinbox.dev
// (a pages.dev preview, localhost) it does nothing.
(() => {
	const id = document.currentScript?.dataset.id;
	const host = location.hostname;
	if (!id || !window.CookieConsent || !(host === 'spinbox.dev' || host.endsWith('.spinbox.dev'))) return;

	window.dataLayer = window.dataLayer || [];
	function gtag() {
		// biome-ignore lint/complexity/noArguments: gtag reads an Arguments object from the layer, not an array.
		window.dataLayer.push(arguments);
	}
	gtag('consent', 'default', { ad_storage: 'denied', ad_user_data: 'denied', ad_personalization: 'denied', analytics_storage: 'denied' });
	window.dataLayer.push({ 'gtm.start': Date.now(), event: 'gtm.js' });
	const tag = document.createElement('script');
	tag.async = true;
	tag.src = `https://www.googletagmanager.com/gtm.js?id=${encodeURIComponent(id)}`;
	document.head.append(tag);

	// The container listens for cc_consent_update, as its own banner sent it.
	const update = () => {
		const analytics = window.CookieConsent.acceptedCategory('analytics') ? 'granted' : 'denied';
		gtag('consent', 'update', { analytics_storage: analytics });
		window.dataLayer.push({ event: 'cc_consent_update', cc_analytics: analytics, cc_marketing: 'denied' });
	};

	// The banner follows the site's theme: dark unless the page says light.
	const root = document.documentElement;
	const theme = () => root.classList.toggle('cc--darkmode', root.dataset.theme !== 'light');
	theme();
	new MutationObserver(theme).observe(root, { attributes: true, attributeFilter: ['data-theme'] });

	const privacy = '<a href="https://blog.spinbox.dev/privacy/">Privacy policy</a>';
	window.CookieConsent.run({
		revision: 0,
		cookie: { name: 'cc_cookie', domain: 'spinbox.dev', expiresAfterDays: 182 },
		guiOptions: {
			consentModal: { layout: 'box', position: 'bottom right', equalWeightButtons: true },
			preferencesModal: { layout: 'box', equalWeightButtons: true },
		},
		onConsent: update,
		onChange: update,
		categories: {
			necessary: { enabled: true, readOnly: true },
			analytics: { autoClear: { cookies: [{ name: /^_ga/ }, { name: '_gid' }] } },
		},
		language: {
			default: 'en',
			translations: {
				en: {
					consentModal: {
						title: 'We use cookies',
						description: 'Google Analytics shows us which pages get read. It sets cookies only if you accept.',
						acceptAllBtn: 'Accept all',
						acceptNecessaryBtn: 'Reject all',
						showPreferencesBtn: 'Manage preferences',
						footer: privacy,
					},
					preferencesModal: {
						title: 'Cookie preferences',
						acceptAllBtn: 'Accept all',
						acceptNecessaryBtn: 'Reject all',
						savePreferencesBtn: 'Save preferences',
						closeIconLabel: 'Close',
						sections: [
							{
								title: 'Your choices',
								description:
									'Your choice applies to spinbox.dev, docs.spinbox.dev and blog.spinbox.dev. You can change it at any time from the "Cookie preferences" link in the footer.',
							},
							{
								title: 'Strictly necessary',
								description: 'Required for the site to work, and to remember this choice. They cannot be disabled.',
								linkedCategory: 'necessary',
							},
							{
								title: 'Analytics',
								description: 'Help us understand how visitors use the site (Google Analytics). Data is aggregated.',
								linkedCategory: 'analytics',
							},
							{ title: 'More information', description: `Questions about our cookie policy? Read our ${privacy}.` },
						],
					},
				},
			},
		},
	});
})();
