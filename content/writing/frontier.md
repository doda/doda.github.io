---
title: "A Few Months at the Frontier"
date: "2026-01-07"
summary: "Lessons from building infrastructure projects with long-running coding agents."
readTime: true
autonumber: true
math: false
showTags: true
hideBackToTop: false
tags: ["ai", "infrastructure"]
---

Over the past two months, partially inspired by this [Anthropic blog post](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents), I've been playing with a long-running coding agent harness. Affectionately called [Looper](https://github.com/doda/looper). 

It maintains state in `.git` via simple files such as `task_list.json` (outstanding tasks) and `agent-progress.txt` (what it did in the last session). Each session, it pops the best next task from the list and attempts to make progress on it. Everything runs in [Modal Sandboxes](https://modal.com/docs/guide/sandboxes).

The initial `task_list.json` is generated from an extensive `SPEC.md` , I used ChatGPT 5.2 Pro to generate these (with a few review cycles). [Example for the `dray` SPEC](https://gist.github.com/doda/25d0ea0c279b43eb21192ba5ac66c2b5).

The motivation behind this was to explore the current capabilities of frontier models to implement complex, well-defined infrastructure projects:

- [Dray](https://github.com/doda/dray) - a leaderless, object-storage based Kafka broker implemented in Golang, that compacts records to Iceberg/Parquet (inspired by WarpStream/Ursa Engine)
- [Vex](https://github.com/doda/vex) - an object-storage based vector search engine (inspired by turbopuffer)

Some of the things I learned during this period:

- Opus 4.5 is smart, incredibly versatile and great at instruction following, but sometimes struggles tackling complex problems, partly due to its shorter context window. It’s great for quickly interactively debugging issues. A few months ago, the stuff I was good at, I was strictly better at than the frontier models, playing with Opus 4.5 during release made me doubt that.
- Codex 5.2 is much slower, but especially on `high` and `xhigh` can tackle very tricky bugs, completely autonomously. It’s honestly beautiful to watch it debug a complex, bespoke data pipeline (such as for the [Looper Demo](https://demo.doda.co/)), and bit-by-bit disprove hypotheses by reading source code, inserting logs, or writing helper scripts.
- Issues encountered:
    - **Context management:** This is probably the #1 thing, the longer context gets, the worse performance is. But the more complex the codebase, the more context you need to make effective changes. While frontier models are very good at building, especially from scratch, this still meant that a single file or instruction missed can result in non-sensical output.
    What helped: Ensuring the model is aware of relevant docs such as “This task references part X of the `SPEC.md` ”, encouraging it to read `agent-progress.txt` as to not go in loops. Actively iterating on prompts.
    - **Cheating / Shortcuts:** From having read way too many think tokens/traces over the past months it is my supposition that the frontier labs inject “tokens remaining: XYZ / time spent: XYZ” messages into message context to keep model outputs economical. Unfortunately these injected reminders also make the model come up with some wild mental gymnastics amongst which I’ve seen:
        - Implementing a new feature, writing tests, seeing them fail and then replacing usage of the new feature with a `Mock` to make tests pass. (Future Claude can deal with this)
        - Marking a task that tries to establish API equivalency as passing, because both the reference implementation and candidate implementation are failing in the same way (500 status code).
        - Very relatably promising to tackle issues not now but “in the next PR” (Narrator: “There was no next PR”)
        - `Mock` implementations everywhere.
        
        What helped: Implementing a “code review” flow, where the same or (preferably) a different model reviews the implementation of the first model. Giving negative examples in prompt of what *not* to do.
        
    - **Tools:** One great trick to making models faster and more reliable is giving them great tools! As an example: For the single-VM deployment of the [Looper Demo](https://demo.doda.co/) I started with [overmind](https://github.com/DarthSim/overmind), but saw the agent struggling to use it (even after adding more instructions). Switching to `systemd` helped here just by virtue of how much training data there must be how to use it.
    - **Prompting:** Still something that is slept on by probably 90% of developers. Whenever your agent makes a mistake, add instructions to `AGENTS.md` / `CLAUDE.md`  to help guide it in that situation.
    - **Multi-Threading / Coordination:** Agents have high latency (slow) but potentially infinite throughput (launch many), so lured by the premise of VIBE CODING ALL THE THINGS, I made `looper` multi-threaded. What ensued was a mess of coordination / thrashing, such as 2 agents fighting over Go versions and continuously attempting to change it (Agent A: Go 1.25 is wrong, the spec says 1.23, Agent B: We have 1.23 but we need 1.25 for this library version with this feature…). Not only that but following agent progress / tracking bugs and improving the harness became a lot more difficult, so I reverted to single-threaded execution.
    - **Overly Defensive:** A pet peeve of mine, this became better with prompting, but the models still love to do purely additive coding, with “safe” defaults, extensive try/except chains and hate deleting code.
    - **System integration:** Even with the most beautiful, extensive SPEC, coding agents still only output new code token-by-token, and reliable, performant systems are forged in the hells of `prod`. After Looper marked all tasks complete, I still made 33 manual commits to dray and 15 to vex. Things that were missing: CI/CD setup, real external system integration (Oxia, Iceberg+DuckDB compatibility), performance optimization, edge case bugs (offset=0 fetch, S3 cancellation), and operational tooling (rebuild scripts, GC workers). Having a completely external user/demo/integration is a great way to keep the system honest.

## The Centaur Age

Where does this leave human developers? We can look at chess, which went through three phases:

- **Human reign** (until ~2006): Deep Blue beat Kasparov in 1997, but humans stayed competitive until about 2006, when Kramnik lost to Deep Fritz.
- **Centaur reign** (2006–2013): Human+engine teams outperformed either alone.
- **Computer reign** (2014–): Engines are strong enough that humans add more noise than signal.

We are currently in the Centaur phase of coding. Coding is different than chess in that it is more open-ended, creative, costly, risky and tends to evolve with changing requirements, etc. So, depending on the domain, humans will likely always remain involved. For more constrained tasks such as competitive programming, we have likely already entered the “Computer reign”.

## Conclusion

Did I manage to build production-grade infrastructure projects, comparable to a [WarpStream](https://www.warpstream.com/) or [turbopuffer](https://turbopuffer.com/)? Decidedly, **No**! But that was never the goal. I would say, however, that I managed to, build robust alpha-stage software for both, in a fraction of the time it would have taken me otherwise. Given that I was partly working on these, in the evenings, while on vacation in Costa Rica, and in parallel, I’d say my output was 10x higher than otherwise.

I’m confident that, given more time investment, the software would mature to a point where it is competitive with these products that have been built over years (the alpha release taking more than a year in each case).

My takeaways from a few months spent at the frontier of coding models (or at least what’s publicly accessible):

1. I’ve been coding for 15 years and I’ve never seen my profession change so dramatically like it has this past year:
    1. Writing tests by hand: Out
    2. Writing code by hand: Mostly Out (where you can afford to)
    3. Writing docs: Out
    4. Bikeshedding: Out
    5. Line-by-Line Code review: Mostly Out (Agents are better and faster for the low-level, the focus is now on high-level design)
    6. Being attached to code: Out (Code has become ephemeral)
    7. Arguing about technical decisions in a vacuum:  Out (Claude built prototypes of each approach over night)
    8. Reading code: Mostly Out
    9. Bespoke/arcane tooling: Mostly Out (Models perform best at tasks that are in-distribution)
2. On the other hand:
    1. Greenfield projects: In (Agents thrive in smaller, modern, standardized codebases)
    2. Prompting: In (Being precise, terse and thoughtfully guiding the models)
    3. Designing verification: In (If it’s verifiable the agent will be able to do it sooner or later)
    4. Multi-tasking: Mostly In (The human brain still struggles with in unfortunately)
    5. Having experience / good intuition: In
3. If the task is verifiable, the agent can do it. 
4. Agent tooling is still in its infancy, models are still very hobbled by a lot of the development tooling being built for humans, this will only get better.
5. I can hardly get excited about writing code anymore, it feels like a thing of the past. The future is in learning how to recursively improve and leverage coding agents.
6. The "last mile" is predictable: ~30% of real-world work falls outside what SPECs capture—CI/CD, external system integration, performance tuning, edge cases, operational tooling. Future SPECs need an "Operations" section.
7. It's time to build. 💪