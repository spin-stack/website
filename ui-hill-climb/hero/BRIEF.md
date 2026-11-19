# The landing page's hero: the brief

What an agent is told to make the hero, as it stands after the prototype in `index.html` beside
this file. Read `CLAUDE.md` and `README.md` at the repository's root first: their rules hold for an
animation as for a page.

## The role

A creative director and motion designer for a technical product's site, at the level of Stripe or
Linear, working in this repository (Astro Starlight).

## The goal

Replace the hero's art (`src/components/Hero.astro`, `src/components/Disk.astro`) with a film that
leaves one idea behind:

> One click, and in my own AWS account a whole machine is put together for me, layer by layer,
> and I am root in it from my browser.

A shot that does not serve that is cut.

## Who watches

Infrastructure and platform engineers. They distrust marketing; what stops them is the real
mechanism, drawn well. A metaphor that has to be explained loses them: the blog tried three
illustration styles for its headers and dropped all of them for that reason. Decide what each
shot says before deciding how it looks. No Matrix rain, no robots, no glowing brains.

## The film

One timeline, about 34 s, that plays by itself from start to end, once. Times are the prototype's.

**The click (0-5 s).** A laptop; in its browser the dashboard's *Create workspace*; the cursor
clicks *Create*. The click becomes a line of light that the camera follows along a fibre to a region
"in your AWS account", into a hall of racks, where one server's slot lights up: a runner with room,
or one started for it. The camera goes into the slot; light fills the screen.

**Your account (5-7 s).** One region of the account, its outline dashed on the floor, and in it
three kinds of machine, **each a machine of its own**, standing apart:

- the **proxy**: the only machine the internet reaches;
- the **control plane**, with its **database** on it, and a panel of what should run;
- a **runner**: a large machine that runs many workspaces, drawn as an open chassis so what runs
  in it shows: its cores and memory, its NVMe (the local copy of the layers), the runner process,
  and other workspaces, each in a jail of its own;
- and the **bucket**.

Nothing says whether these share a rack or a zone: no rack, no availability zone, is drawn
around them.

**01 Layers.** The runner connects to the control plane (the line is drawn from the runner: it
opens every connection it has) and the whole list of what should run on it comes back. The layers
the runner does not have leave the bucket, pass a ring that turns green (checked against a digest,
decrypted), light the NVMe and stack up.

**02 Clone.** One new, light layer drops on top: every write lands there; the image under it is
read-only and shared; "bytes copied 0".

**03 Boundary.** QEMU's jail is drawn edge by edge around the disk: a user of its own, nothing
opened by name, a network namespace of its own, empty. The preset's cores and memory light on the
runner; the disks, sockets and `/dev/kvm` go in from the runner process already open.

**04 Boot.** The serial console types the boot in order, and each line drives the room: `vminitd`
ready (the kernel lights), the hand-off to the workspace's own init, the supervisor's config over
vsock (a dashed line to the runner process, "not the network"), its disk, the connection to the
control plane, the SSH certificate authority, the trust store's egress root, `login:`. The host's
egress is a gate at the runner's edge, closed: one request outside the policy stops at the bars
and turns them red, one a policy allows goes through a lane that opens, green. At `login:` the
runner opens its connection to the proxy's relay: none is ever opened to a runner.

**05 Connect.** On the laptop, the workspace's page in the dashboard: `api` RUNNING, and *Open
terminal*. The click goes over HTTPS to the proxy, through the relay the runner had connected to,
to the runner and into the machine; the browser's terminal shows `# whoami`, `root`.

**06 Running.** Everything else fades, and the disk becomes the mark.

## The mark

`logos/mark-clear-512.png` in spin-stack/brand, modelled, not approximated: a rounded blue bowl
(blue at the rim to deep at the base), a navy groove, a cyan-to-mint disk that bulges past the
bowl, a navy groove, a frost top with a raised rim, a swirl that starts thin at the back left,
wraps round the front and runs into the middle disk (a thinner stroke outside it), and its shadow.
The light comes from the left. The prototype builds it from lathed profiles with vertex colours
and a tapered tube; the film's stack (three image layers and the workspace's layer) turns into it.

## What is true, and where it is said

Every label and caption says only what a page of the site already says, and links to it ("How we
know"). The film's facts and their pages:

| What the film shows | The page |
|---|---|
| Proxy, control plane, runners, database, bucket; the proxy the only machine the internet reaches; the relay for SSH, a terminal or a port | `overview/how-it-works` (The pieces) |
| A runner opens every connection it has, to the control plane and to the relay; none is opened to a runner | `overview/how-it-works` |
| The control plane sends each runner the whole list of what should run on it | `overview/how-it-works` |
| Layers fetched from the bucket, checked against a digest, decrypted, a chain of qcow2; the NVMe copy | `overview/how-it-works` |
| A clone copies nothing; the image is shared | `concepts/images` |
| QEMU's user, nothing by name, no network, a namespace of its own | `security/` |
| `vminitd`, the supervisor over vsock | `overview/how-it-works`, `concepts/workspaces` |
| The disk: the image's size plus the preset's free space | `concepts/workspaces` |
| Egress closed until a policy opens it; the host's egress proxy; the egress root in the trust store | `concepts/network`, `use/agent-keys` |
| *Open terminal*, a shell in the browser over the relay | `use/editors` (The dashboard's terminal) |
| A runner with room, or one started for it | `concepts/workspaces` (States) |

And the rules around them:

- The film's words and steps live in a data file with `reads`, as `src/lib/story.ts` does, not
  in the component. A number is a `<Value>` or has its line in `tools/refgen/figures.txt`: the
  film carries none today, on purpose.
- What a real installation shows (the CLI, a console, the dashboard) helps to understand, never to
  copy: ids, addresses, host names and domains stay out, and the console lines are representative,
  in the order a boot prints them.
- No logo or trade dress of AWS or anyone's. "Your AWS account" and "S3" as words.
- A mechanism is described, never vouched for: "closed until a policy opens it", never "nothing
  gets out".
- Nothing of spin's that the pages do not publish: `task exposure` is the check.

## The player

- It plays the whole film by itself, once, and holds the last frame.
- A bar along the bottom: play and pause; chapters (Click, Layers, Clone, Boundary, Boot, Connect,
  Running) that fill as the film plays and jump to one when pressed; Replay.
- **Step by step** is optional: when it is on, the film stops at the end of each chapter and
  *Next* goes on. The arrow keys move between chapters, the space bar pauses.
- It pauses when the hero leaves the screen.
- With reduced motion asked for, it opens on the last frame; without scripts, the hero is a still
  of that frame (the integration's job: the prototype has none).

## The look

The brand's tokens (`dist/tokens.css` in spin-stack/brand, and the site's): the dark page, one
accent (the signal the camera follows), the mint for what the machine says back, green and the
error pink for the egress gate. Lines thin, like a blueprint; a light bloom only on what glows.
IBM Plex Sans and Mono. The labels are HTML over the canvas, so they stay sharp and are read in
the page's own type.

## The technique

- three.js r170 (`three.module.min.js` is 171 KB with gzip; the bloom and the label renderer
  add about 6 KB), loaded after the headline, which stays HTML.
- The canvas is sized in CSS (`width: 100%; height: 100%`): a canvas is a replaced element, and
  `inset: 0` alone left it at its drawing buffer's size, twice the stage on a retina screen.
- Each timeline value is a pure function of the film's time, so a jump, a pause or a step lands
  on the same frame as playing does. `?at=<seconds>` freezes the prototype there.

## What is left

1. **Integration.** `three` into `package.json` (the install goes through `sfw` in the image the
   `Dockerfile` makes, never on the host); `Hero.astro` loads the scene without blocking; the
   words and steps into a data file with `reads`; the still for reduced motion and for no scripts;
   `Disk.astro` and what only it used retired; `task check` green.
2. **The real dashboard.** The browser's two screens (*Create workspace*, the workspace's page
   with *Open terminal*) from the screen catalogue of spin-stack/brand, not drawn by hand.
3. **Not verified yet.** A phone's width (headless Chrome would not go under about 500 px); the
   frame rate on a modest GPU; the light theme; the film playing through in real time (it was
   checked frame by frame with `?at=`).

## Done when

Someone who has never heard of Spinbox watches it once, without sound, and can say: "it makes a
real machine in my account, from a disk of layers that are not copied, shut in and with the
network closed, on a machine of its own apart from the ones that run it, and I get a root shell in
my browser through the one machine the internet reaches." If all they can say is "it looks cool",
it failed.
