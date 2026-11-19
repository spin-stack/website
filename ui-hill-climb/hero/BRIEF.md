# Spinbox hero animation brief

Finish and integrate the Spinbox hero animation in this Astro Starlight repository. Before editing,
read `AGENTS.md`, `README.md`, `flow.html` beside this brief, and `src/components/Hero.astro`.

## Goal

After one viewing without sound, a visitor understands that Spinbox puts a whole machine in their
own AWS account, built from layers that are not copied, shut in and with its network closed, on a
machine of its own apart from the ones that run the installation, and that they reach it - a
terminal, a command, SSH - through the one machine the internet reaches.

## The form

A live system diagram, not an illustration and not a camera move: one frame, the whole
installation in view from the first second, and only the data moving. The reference is a diagram
in the style of a product's own interface: cards, mono labels, curved wires, labelled particles
travelling along them, counters that rise and states that change, a legend that says what a
particle is, a progress line. What makes it believable is specificity: real names, real states,
real commands. Photorealism and metaphor were both tried and dropped: a 3D film read as a tech
demo, and a short cut of a disk and a terminal lost the parts that make the claim credible.

`flow.html` is the implementation base: its layout, its timeline and its words.

## The frame

Four columns, the runner the widest:

1. **Who connects** - the dashboard in a browser, an agent with the CLI, an editor over SSH.
2. **Two small machines** - the control plane (API, dashboard, PostgreSQL; what should run, by
   workspace and runner) and the proxy (Caddy on :443, the only machine the internet reaches:
   dashboard, ports, relay).
3. **A runner, a large machine of its own** - the runner process; the workspace `api` as a jail
   (QEMU) with its disk drawn as layers; two other workspaces, each a jail of its own; the host's
   egress.
4. **The bucket** - the layers, each with its digest; later, the workspace's sealed layer. And
   below it, the internet, as far as a policy names it.

## The timeline (about 24 s)

| Time | What moves |
|---|---|
| 0.9-2.2 | `create api`: browser → proxy → control plane; `api → runner 1` goes `CREATING` |
| 2.8-4.2 | The runner opens its connection to the control plane; `list` comes back; `STARTING` |
| 4.6-7.1 | `L1`, `L2`, `L3` from the bucket, each ✓ against its digest; the shared layers fill |
| 7.6 | The workspace's own layer; bytes copied 0 |
| 8.6-9.3 | The jail: a user of its own, nothing opened by name, a namespace of its own, empty |
| 9.8-12.2 | The serial console: `vminitd`, the supervisor over vsock, the control plane, `login:`; `RUNNING` |
| 14.0-15.4 | The host's egress: one request outside the policy stops (denied 1), one passes (allowed 1) |
| 15.6-16.2 | The runner opens its connection to the relay |
| 16.6-20.7 | Three sessions through the relay: `T` Open terminal → `# whoami` → `root`; `E` `spin exec api -- make test` → ok; `S` `spin ssh api` → a prompt |
| 21.6-22.6 | The workspace's layer is sealed and uploaded: the bucket gains a row |

## Claims and sources

Every visible word is either a name the example invents (`api`, `web`, `build`, `runner 1`) or
something a published page says. Keep the film's words and steps in a data file with `reads`,
following `src/lib/story.ts`.

| What the diagram shows | Published page |
|---|---|
| Proxy, control plane, runners, database, bucket; the proxy the only machine the internet reaches; the relay for SSH, a terminal or a port | `overview/how-it-works` |
| A runner opens every connection it has; none is opened to a runner; the control plane sends the whole list | `overview/how-it-works` |
| Layers fetched from the bucket, checked against a digest only the control plane writes, decrypted | `overview/how-it-works` |
| A clone copies nothing; the image is shared | `concepts/images` |
| `CREATING`, `STARTING`, `RUNNING` | `concepts/workspaces` (States) |
| QEMU's user, nothing by name, no network, a namespace of its own | `security/` |
| `vminitd`, the supervisor over vsock | `overview/how-it-works`, `concepts/workspaces` |
| Egress closed until a policy opens it | `concepts/network` |
| Open terminal, `spin exec`, `spin ssh` | `use/editors` |
| A layer sealed and uploaded to the bucket | `overview/how-it-works` (Where your work is stored) |

No number appears that a page does not give: a counter counts what the diagram itself shows
(sessions, requests), never sizes or times. Keep installation identifiers, addresses, hostnames
and domains out. A mechanism is described, never vouched for.

## Composition and integration

- The diagram is a full-width element: it takes the hero's width under the headline, or a section
  of its own right after it. Decide with the owner before integrating.
- Under about 900 px wide, the columns stack: who connects, then the two small machines, then the
  runner, then the bucket; the wires become short vertical joins, and the particles still travel.
- Build it from HTML and SVG, as `flow.html` is: no WebGL. Its text is real text, so it scales
  and stays sharp; the colours are the brand's tokens.
- The final frame is the static state: without scripts, with reduced motion, and while it loads.
- Play once when it comes into view, pause out of view, hold the final frame. Pause and Replay,
  keyboard-reachable. Keep `?at=<seconds>` in the prototype for reviewing frames.
- Use the screens of the brand catalogue for the dashboard's two moments if they read at this
  size; otherwise keep the simplified card and record the substitution.

## Verification and delivery

Play it through in the Astro page on desktop and on a phone; light and dark themes; pause and
replay; reduced motion; no scripts. Run `go tool task check`. Deliver the changed files, one
screenshot per row of the timeline, and what is left unverified.

Done when someone who has never heard of Spinbox can say, after one viewing: "it makes a real
machine in my account, from a disk of shared layers, shut in and with the network closed, on a
machine apart from the ones that run it, and I get a root shell in my browser through the one
machine the internet reaches."
