---
title: "Engineering Principles"
date: "2023-11-12" # Assuming today's date, please update if needed
summary: "A collection of engineering principles."
description: "A collection of engineering principles."
readTime: true
autonumber: true
math: false # Assuming no math content, please update if needed
tags: ["engineering", "software"] # Suggested tags, please update if needed
showTags: false
hideBackToTop: false
---

## Working with Others

- **Software is about People**
  At the end of the day, meaningful software only gets written by teams, made up of people, and exists primarily to solve people's problems. Master your craft yes, but most importantly endeavor to be great to those around you, and those you serve.

- **Saying Yes / Saying No**
  I'm a big fan of personality psychology and the [Big Five](https://en.wikipedia.org/wiki/Big_Five_personality_traits). While for most of them you generally want to be on one end, with Agreeableness, you generally want to be somewhere in the middle. Too agreeable and people think you're a pushover, too disagreeable and people will think you're harsh and cold. Being polite and positive is generally highly regarded in American culture, so being able and willing to effectively say No is a highly valuable skill.

- **Be Concise**
  Great engineers are concise. Rather than writing long essays talking you through their thought process, they tend to convey just the minimum amount of information necessary to get their point across.

- **Overcommunicate**
  In an [old blog post](https://web.archive.org/web/20130116014845/http://blog.alexmaccaw.com/stripes-culture) about Stripe's early work culture Alex MacCaw described Stripe's internal comms as a "totally transparent", "fearless" and a "fire hose". When in doubt, communicate, and publicly!

- **Beware the emotional bank account**
  Every technical argument has attached to it a social and emotional cost. Before jumping in, consider if it's worth it, or if you can just let it slide. Some ways to deposit into the emotional bank account are praise, quick code reviews and tackling things that are technically "not your job".

- **Code rules supreme**
  When having a technical debate, just getting some code, like an example, or a prototype, or a fix in front of people can usually be enough to sway people. The best comments in Slack discussions are often just a single GitHub PR link.

- **Stand up for Technical Health**
  At times fellow engineers get frustrated with all the "annoying" product people / devs constantly hacking new features into the codebase and breaking things. One great way of reframing this that I've read in [An Elegant Puzzle](https://www.amazon.com/Elegant-Puzzle-Systems-Engineering-Management/dp/1732265186) is that as an Engineer(ing Leader) it is your job to stand up for and argue for the technical health of the application you are working on. Just like it is for Product to come up with and get you to implement new features!

- **Be Right, A Lot**
  This is shamelessly stolen from Amazon's [Leadership Principles](https://www.amazon.jobs/content/en/our-workplace/leadership-principles) but I want to highlight it again. "Leaders are right a lot. They have strong judgment and good instincts. They seek diverse perspectives and work to disconfirm their beliefs." I've found that among the best engineers there is usually very little disagreement once the facts are well understood.

---

## Working with Yourself

- **Work like a lion (via [Naval](https://www.youtube.com/watch?v=aLhF01_nJ9Q))**
  As a modern knowledge worker you train hard, you sprint, then you rest. It's completely OK, and even desirable to move like a sloth if that allows you to then switch into a breakneck pursuit.

- **Develop a "killer instinct"**
  I don't recall where I read this, but one mark that makes the "greatest scientists" is that they developed a "killer instinct" where they could parse out singular threads from a wide field of opportunities and pursue them relentlessly. I think a similar thing applies in engineering where you are almost more defined by the things you don't do than the ones you do.

- **Read papers and engineering blogs**
  This has been — bang for my buck — one of the biggest uplevels in terms of system design. Some of the most brilliant engineers of our time discuss how they solved tricky engineering problems at scale. It dispels the magic around complex software, and gives you proven building blocks for building your own solutions. Our Real-time infrastucture at Substack was directly inspired by this [FB tech talk](https://www.youtube.com/watch?v=ODkEWsO5I30) which I had watched 2 years earlier. The idea and knowledge lingered in my brain until the time came.

- **Leave problems unsolved**
  Sometimes when there is a niggle, or something annoying, it can be good to leave that "problem" unsolved. Tactical procrastination allows for that problem space to either 1. stop being a problem 2. morph or 3. become more well defined. By delaying implementing a solution you also allow yourself to gather other experiences that better inform your solution, technology to further evolve and your subconscious to keep gnawing on the problem.

- **Be bold**
  One thing that has helped me be such an effective infrastructure engineer is to be bold. If you never break things you are moving too slowly. In other words, there is some optimal error rate you want to be operating at and it's not 0. Sometimes you are about to roll out, say a Node.js version upgrade across your cluster, you can either empirically test it in every which way, or trust your intuition and yeet it out. Canary deployments help.

---

## Working with Infra

- **If it's not in CI it might as well not exist**
  Devs love to shipit 🐿️. If you want to enforce a constraint on your codebase, the only way to do that is if it is enforced by CI.

- **You won't do it later**
  I got this from a [DHH](https://dhh.dk/) blog. The gist is that people sometimes ship a feature and leave some important part of it as a "I'll do it later". You'll never do it later. Stop lying to yourself and others. Embrace shipping things quickly and owning that, or do it right from the get-go.

- **[Rule of least power](https://en.wikipedia.org/wiki/Rule_of_least_power)**
  Choose the least powerful language or system suitable for a given task. For example, prefer static HTML over a complex JavaScript framework if all you need is to display static content. This often leads to simpler, more robust, and more maintainable systems. The best code is no code!

- **Design systems simply**
  One thing I've learned by consuming way too many [aviation](https://admiralcloudberg.medium.com/) [accident](https://www.youtube.com/@MentourPilot/videos) [analyses](https://www.youtube.com/@blancolirio/videos) is that designing systems that are effective yet safe for operators is surprisingly difficult! Operator cognitive capabilities generally drop a few dozen points during incidents and so it is important to design systems simply. "Magic" and "complexity" that work well in 99% have a tendency to blow up in really bad and surprising ways in the 1% of cases.

- **Staging environments are the bee's knees**
  Having a (relatively) complete replica of your production setup in a separate AWS account is wonderful and allows you to test the riskiest of changes in a safe way.

- **Overprovision**
  A lesson learned the hard way — being frugal is good — but being online is better, especially if you can afford it 😅 Do your best to estimate the predicted load of new systems, but in general you'll want to include some [Factor of Safety](https://en.wikipedia.org/wiki/Factor_of_safety) in case 1. there's a sudden spike 2. you made a mistake 3. AWS ElastiCache decides that exceeding the network allowance for your cluster for a few minutes is grounds to promptly traffic shape you and your friends into a SEV-1 outage.

- **Know the good parts of your cloud**
  Inspired by Daniel Vassallo's [The Good Parts of AWS](https://dvassallo.gumroad.com/l/aws-good-parts), AWS has certain "core services" that are battle-tested and great building blocks. Use them freely and liberally!

- **Serverless is fake**
  Lambda, Fargate and other serverless technology promise freedom from "servers". "no infra management", "pay only for what you use", "automatic scaling" and "prod now". But in reality there is always a server! It is just abstracted away from you. For a lot of use cases that is _fine_ or even desirable, but when you get to running things at scale, or for a large number of use cases or developers, you will generally run into the limits of some of these and wish you could modify the layers beneath you.
