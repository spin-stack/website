// Google Analytics 4 with Consent Mode v2, loaded by Head.astro only when the build has a
// measurement ID (data-id). Every kind of consent is denied before gtag.js loads, so it sets no
// cookie and sends only the cookieless pings Consent Mode allows. A file, not an inline script,
// so the Content-Security-Policy names it with 'self'.
const id = document.currentScript.dataset.id;
window.dataLayer = window.dataLayer || [];
function gtag() {
	// gtag.js reads an Arguments object from the layer, not an array.
	window.dataLayer.push(arguments);
}
gtag('consent', 'default', {
	ad_storage: 'denied',
	ad_user_data: 'denied',
	ad_personalization: 'denied',
	analytics_storage: 'denied',
});
gtag('js', new Date());
gtag('config', id);
const tag = document.createElement('script');
tag.async = true;
tag.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(id)}`;
document.head.append(tag);
