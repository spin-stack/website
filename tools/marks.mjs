// The header draws the mark at the nav's height, and src/assets/logo*.png are 256 px copies of
// spin-stack/brand's mark that tools/refgen/brand.go compares byte for byte. So each build derives
// a small WebP beside them, a few kB instead of 42, and leaves the copies as they are.
import sharp from 'sharp';

for (const name of ['logo', 'logo-light']) {
	await sharp(`src/assets/${name}.png`).resize(96, 96).webp({ quality: 90 }).toFile(`src/assets/${name}.webp`);
}
