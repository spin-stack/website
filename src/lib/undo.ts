// The session the landing page's hero types (Undo.astro) and its social card shows (og.png.ts):
// a snapshot, the whole disk removed from inside, the machine back with its processes. What each
// command prints is what spin v20261118.01 printed on a workspace made for it (the snapshot and
// the restore took 2.4 s and 1.6 s); rm's complaints about /sys and /proc are cut to one.

// What the hero claims, each a row of spin's feature board that an end-to-end test checks
// (task reference:check holds them): the machine is the Bento's to say, the diff too.
export const claims = [{ feature: 'Snapshots with memory, resumed with their processes' }, { feature: 'Restore a workspace to a snapshot' }];

export type Line = { cmd: string } | { out: string; tone?: 'ok' | 'bad' | 'dim' };

const snap = '01a1fb7c-970e-72d0-a333-635af5a6b0a2';

export const session: Line[] = [
	{ cmd: 'spin snapshot create api --with-memory --wait' },
	{ out: `Snapshot ${snap} published`, tone: 'ok' },
	{ cmd: 'spin exec api -- rm -rf --no-preserve-root /' },
	{ out: "rm: cannot remove '/sys/module/sctp/parameters/no_checksums': Operation not permitted", tone: 'dim' },
	{ cmd: 'spin exec api -- ls' },
	{ out: '/bin/bash: No such file or directory', tone: 'bad' },
	{ cmd: `spin workspace restore api ${snap} --with-memory` },
	{ out: `Workspace api resumed from snapshot ${snap}`, tone: 'ok' },
	{ cmd: 'spin exec api -- ls /' },
	{ out: 'bin  boot  dev  efi  etc  home  init  lib  media  opt  root  run  srv  usr  var  workspace' },
];
