# frogly

frogly is a task planner for self-organized teams. You host it yourself.

Most planning tools are built for the people who watch the work. They ask
for fields, reports and rituals, and every idea becomes a ticket the moment
someone writes it down. Boards fill up, nobody tends them, and the team
stops trusting them.

frogly starts from the other end: how an idea arrives, and how little it
should take to write it down. Using it should make you think "why isn't
everything like this?". Then you just use it.

## How it feels

Calm. Lots of space, nothing that shouts, no red badges, no urgency the
team didn't choose. frogly takes its name and its mood from Bashō's haiku:

    古池や　蛙飛び込む　水の音
    furu ike ya / kawazu tobikomu / mizu no oto

    old pond
    a frog jumps in
    the sound of water

Matsuo Bashō, 1686. The English translation is our own.

A small thing drops in, makes one clear sound, and the surface goes still
again. That's what writing down an idea should feel like.

## How work flows

Ideas grow in a garden. Once enough people want an idea, it's ripe, and
someone harvests it: it becomes a ticket on the board. The team works the
ticket until it's done. Then kaizen, Japanese for "change for the better",
is how the team looks at how its work flowed and plants ideas to improve.

```
garden ──harvest──▶ board ──▶ done
  ▲                             │
  └──────── kaizen ◀────────────┘
```

### The garden

Every project has a garden of ideas. A project is any area of work the
team chooses: an app, hiring, an office move.

You plant an idea with a line of text. That's all. frogly records who
planted it, when, and where it came from: the web, the command line, an
email, a form or an agent. A source like a repository or an email address
is linked to a project, so you rarely have to pick one.

Ideas don't have to be ready. They can sit in the garden and ripen:

- When someone plants something similar, frogly suggests adding it to the
  existing idea as a "me too". The number of different people behind an
  idea is its ripeness, and ripe ideas rise on their own.
- Related ideas can be grouped loosely into a bed, like "everything about
  search".
- Ideas nobody touches fade slowly, like ink. Fading follows the project's
  activity, not the clock, so a quiet project keeps its garden as it is.
  A fully faded idea goes to the compost. It stays searchable, and it comes
  back when someone plants something similar.

### Harvest

A ripe idea becomes a ticket, or it's dropped with a one-line reason.
Dropped ideas go to the compost with their reason, so the team never has to
wonder "didn't we decide this already?".

Nobody has a triage duty. When someone pulls work and the ready list runs
low, frogly shows them the ripest ideas first. One tap turns an idea into a
ticket. The ticket goes into ready at a place its ripeness suggests. If
nobody pulls work for a while, the weekly digest offers one ripe idea to
one person, taking turns.

### The board

The board is boring Kanban, on purpose. Kanban is good because it's
boring.

Every board starts with ready, doing and done. A team adds its own columns
between ready and done when it needs them, like "review", each with its own
WIP limit, the most tickets the column may hold at once. Ready is an
ordered list, and the top is next. You pull a ticket
and it's yours.

A ticket has a key like `FROG-12`, a title, a description, the ideas it
came from, one owner, a checklist, an optional due date and an on-hold
flag. Work that repeats is a ticket that comes back on a schedule. Nothing
else. A new field has to earn its place, and one comes out in return.

### Kaizen

frogly knows when each ticket entered each column. From that it shows the
team how its work flows: how long tickets take from ready to done, which
work is getting old, how long things wait. Ideas for working better are
planted in the garden like any other idea, and the evidence shows whether a
change helped. The numbers are for the team, never a report for a manager.

## Your day

frogly opens on today. At the top is the box to plant an idea. Below it is
what's yours across all projects: what you're doing, work someone handed
you, due dates coming up, and ripe ideas when a ready list runs low.

Only a person can interrupt you. The one notification that comes right
away is someone handing you work. Everything else waits for a weekly
digest in the team's chat or email: what shipped, the ripest ideas, how
work flowed, what got stuck and what went to the compost.

## Agents

An agent is anything that uses frogly's API for a person: an AI assistant
like Claude, the command line, a script. It can do whatever its person can do,
including planting, harvesting and finishing work. frogly records how
everything happened, like "done by Karo via Claude". Ripeness counts
people, so ten agents working for the same person count as one voice.

frogly uses AI only where it helps, only locally and cheaply. A small model
on the frogly server notices when ideas look alike and suggests "me too"
and beds. People decide. Without the model, frogly still works.

## Principles

- Boring but innovative. Boring wherever boring works, innovative only
  where it makes frogly different: the garden and kaizen.
- Each concept appears when it's needed, not before. A new team sees one
  text box on a still page.
- Only a person can interrupt you.
- Ideas and tickets are different things.
- Priority comes from evidence, not from a field.
- The tool does the tending. Nobody's job is to watch the board.
- A field has to earn its place.
- The team improves its own way of working, with evidence.

## How we know it works

We plan frogly in frogly. It works when we use it every day and it feels
good.
