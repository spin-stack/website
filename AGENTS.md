# AGENTS.md - website

The documentation of Spinbox, an Astro Starlight site. `README.md` is the catalogue: every check
the site keeps, and what each reads. This file is what the checks cannot say.

**A page is generated from the code it describes, or held to it by a check, or it does not
exist.** Everything below follows from that.

- **A page with `source:` in its frontmatter is written by `task reference`**: everything under
  `src/content/docs/reference/`, and the pages that are a document of spin's or of the module's
  (`spinDocs` in `tools/refgen/spindocs.go`, `included` in `tools/refgen/include.go`). Edit the
  source, in its repository; an edit to the page is undone by the next run.
- **Prose spin already keeps is not written again here.** A page that would restate one of spin's
  documents becomes that document (`spinDocs`), and the others link to it. Reading every page
  against one release found some thirty sentences the sources no longer said, and nearly all were
  paraphrases.
- **Each kind of sentence on a hand-written page has the form a check reads.** What Spinbox does
  is a link to the board's row, `[words](<feature:the row's name>)`. A duration, a size or a count
  is a `<Value>`, or has a line in `tools/refgen/figures.txt` naming the constant that decides it.
  A command is in a `bash` block. A sentence in none of these forms is held only by the page's
  `reads`, which is the weakest: prefer the form.
- **A page's `reads` are the files whose change would make it wrong**, the narrowest that do: a
  file, never a package. When `task reference:check` says one changed, run the `git diff` it
  prints, read the whole page against it, correct the page, and only then `task reviewed
  PAGE=<the page>`. A stamp says somebody read the page; one put there to turn the check green
  is the failure this exists to prevent.
- **A finding is checked against the source before a page changes.** An audit by an agent is
  right about half the time: open the file it cites and read the line. The site's own checks
  have holes too - one took a `<placeholder>` for a redirection and passed a flag the CLI does
  not have - so a check that passes is not a page that is true.
- **A feature spin ships reaches a page, or `tools/refgen/unsaid.txt`.** The list is what the
  site owes its readers; it shrinks by pages being written, and a line is added only for a row
  the same pull request does not have a page for yet.
- **A new page is in the sidebar's list** (`nav.mjs`, under `src/lib/`), has a `description` of one sentence, and names its
  `reads`.
- **Branch `sync/sources` is written over every day** by `.github/workflows/sync.yml`. A fix to
  a hand-written page that the sync's pull request needs rides that pull request and merges
  before the next run, or goes to `main` first.
- **This repository is public, and a pull request is public the moment it is opened.** spin is
  not: nothing of spin's goes in a commit, a pull request or an issue here but what the pages
  themselves publish. `task exposure` is the check.

| changed | run |
|---|---|
| a page, a component, `tools/` | `task check` |
| nothing here, and the sync's pull request is red | `task check`, and read what it names |
| `versions.yaml` | `task reference`, then `task check` |
| a workflow | `task gate` |

Tasks run as `go tool task <name>`. spin is private: `task sources` reads it with `SOURCES_TOKEN`
(`gh auth token` gives one) or git's own credentials. A host needs Go and Docker: Node and pnpm
run in the image the `Dockerfile` makes, never on the host.

## Claims and tone

The site and the blog (blog.spinbox.dev, where the reasons and the failures live) describe the same
project, and hold it to the same rules. The site says what Spinbox is and how to use it; the blog
says why it is built that way and what went wrong. Link to a post where a page would explain a
reason, instead of restating it: `[why](<post:the-slug>)`, a slug of `src/data/posts.json` (the
published posts; a draft's slug is private and goes nowhere here).

- **What a workspace is comes first; the sandbox follows from it.** A workspace is a whole machine,
  built from parts Linux and QEMU already have, made for a person to work in. That is what makes it
  a place to run code you don't fully trust, an agent's included. Lead with the machine, not with
  "sandbox".
- **No absolute guarantees.** "Nothing reaches the host" is a promise a reader can test and find
  wanting. Say what the mechanism is (its own user, nothing opened by name, a namespace of its own)
  and link to the security model and its known gaps. A boundary is described, never vouched for.
- **Built and tested is not used every day.** A feature with a test behind it is built; say "used
  day to day" only of what is. The agent features are the least lived-in part today: present them
  as ready and tested, not as proven in production.
- **Plain words, no slogans.** Say the verdict, then the reason, in the plainest words that are
  still exact. A number has a source (`<Value>`, `figures.txt`) like any other.
