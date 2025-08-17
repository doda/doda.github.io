---
title: "What I learned at Substack"
date: "2024-08-15"
description: "Six crucial lessons from building and scaling infrastructure that go beyond technical skills: team dynamics, communication, trust, and the human side of engineering."
readTime: true
autonumber: true
math: false
showTags: true
hideBackToTop: false
---

In software engineering, we often focus on the technical side of our work—designing scalable systems, writing clean code, and mastering our tools. But the most impactful lessons are often learned in the spaces between the pull requests. They're about people, trust, and self-awareness. Here are a few hard-learned lessons from my time building and scaling infrastructure.

### Lesson 1: A Technical Win Can Be a Team Loss

When I joined Substack, the company was running on about 40 Heroku dynos. A migration to AWS had been discussed, but three months into my tenure, no real movement had occurred.

Excited by the challenge, I dove in headfirst. Rather than focusing on bringing the team along with me, I started building a prototype of what our web compute could look like on AWS. I wrote a small design doc, got the green light, and then began toiling away at a completely new compute architecture—VPCs, security groups, networking, CI/CD, optimized Docker images, and autoscaling on Fargate.

I look back on those months with a mix of nostalgia and remorse. I was living, breathing, and thinking non-stop about AWS. On evenings and weekends, my mind was consumed with the migration. The result was a technical success, but my extreme work ethic came at a cost. It soured my relationship with two of the more senior engineers on the team.

In Stephen Covey's terms, I wasn't thinking "win-win." I wasn't even thinking "win-lose." I was mostly just thinking "win." My focus was on the technical achievement and, if I'm being honest, on having my name attached to the cloud migration.

I managed to rebuild those relationships after the project was done. But if I could do it again, I would communicate my intentions more openly from the start. I would have done more to involve my colleagues, champion their contributions, and ensure the victory belonged to the entire team.

### Lesson 2: Code Reviews Are Context-Dependent

I used to believe that code reviews were one of the most sacred and important activities for an engineer. They are a tool for ensuring quality, sharing knowledge, and establishing standards. I've since learned that's not always the case.

At Substack, the average experience on my team was over 10 years. In that environment, we all wrote solid code. We trusted each other. We didn't need to nitpick every line; we knew when to reach out for ideas or a critique, and that collaboration often happened organically in Slack. When you have a high-trust team where individuals have clear ownership over parts of the codebase, you can get away with very light-touch, asynchronous reviews. This leads to the next point.

### Lesson 3: Talent Density Is a Force Multiplier

If you can assemble a group of smart, kind, driven, and technically excellent engineers, the dynamic of work changes completely. You can spend your time doing what you love most: building and shipping.

Being surrounded by excellent peers makes you excited to come to work and show progress. It cultivates a no-nonsense environment—no bullshit meetings, no defending the codebase from ill-conceived ideas. The collective focus is on moving forward and building great things. It's honestly a pleasure.

This is another reason why hiring is so important. You want the divas that are a little bit difficult but great at what they do, and you want the people who may be a technically a bit weaker but who excel on soft skills.

### Lesson 4: Trust Your Instincts

This might sound self-congratulatory, but a crucial lesson I learned was to trust my gut feelings about technical and team decisions. Almost every time I had a bad feeling, it was eventually proven right.

There were hiring decisions where I was the dissenting voice, we hired the person anyway, and they turned out not to be a fit. There were technical approaches chosen that I was against, but I allowed the team to convince me, only to watch the predicted problems arise later. The lesson wasn't to become an uncooperative naysayer, but to learn to articulate my instincts more forcefully and to trust my own hard-won experience.

### Lesson 5: Communicate for Your Audience

Having moved to the United States from Germany six years ago, I had to re-learn how to communicate effectively. The cultural norms are starkly different.

In Germany, it's common to disagree vehemently. Complaining about things is a form of small talk. The linguistic scale is different, too. If something is good, it's "okay." If it's bad, it's "terrible."

In the States, if something is good, it's an "amazing idea." If something is bad, it's "something we might want to consider down the line." Learning to navigate these nuances and make my opinion felt in the appropriate manner was critical. The best technical idea is worthless if you can't communicate it in a way your audience can hear.

### Lesson 6: Be Concise

The best engineers I've worked with have a very high signal-to-noise ratio. They convey the minimum amount of information necessary to get their point across, whether in a design doc, a Slack message, or an interview. It's a skill that reflects clarity of thought and respect for others' time.

### Lesson 7: AWS++ Vendors\-\-

We consistently ran into strange issues with various vendors, and became so vendorophobic that by then end we almost always defaulted to implementing system designs to be AWS-native. Exciting queueing solution (eg. NATS) not available as an AWS managed service. It immediately becomes a lot less enticing.

Substack being social-network-adjacent in both its mission and definitely load profile meant that we were very read-heavy (>90%). At peak, our cluster was handling 340,000 hits per second, which made vendor reliability issues particularly painful.

One such moment occurred during an especially annoying outage: We were using Redis Labs (The Original Redis Company and where the creator of redis, antirez, now works again!) to host all our caches. 

One of our developers decided to increase the size of one of our caches from 12GB to 30GB. Unfortunately this also causes the Redis Labs to automatically shard your database. We had affordances in our client code that were equipped to deal with clustered Redis, but that code had suffered some bitrot since we last used it and was not something we wanted to deploy in the middle of an outage. We specifically had some multi-key commands that were triggering `CROSSSLOT` errors.

During the outage Zoom I remarked how stupid it was that we couldn't have a single Redis instance with more than 25GB memory. Fast-forward 6 months and another mini cloud migration later, we managed to migrate our entire caching cluster from Redis Labs to AWS ElastiCache. That also allowed us a few other nice benefits. Such as

1. Valkey 8.0 (including a new asynchronous I/O threading model with 3x higher throughput)
2. Automatic failover
3. IaC tie-in with the rest of our infra, including mature Terraform providers
4. Not having to use Redis Labs' awful dashboard anymore
5. AWS "production-hardened" config where they remove a lot of the potential footguns for you
