---
title: "Building Real-Time Infrastructure at Substack"
date: "2024-08-14"
summary: "How we built a scalable real-time Pub/Sub system that handles a billion messages per week."
description: "Learn how Substack built Zync, a real-time infrastructure system using Node.js and Redis that scales to handle a billion messages per week."
readTime: true
autonumber: true
math: false
showTags: true
hideBackToTop: false
---

### **Introduction: The Need for Real-Time**

At Substack, we wanted to enable chat and other real-time use cases to make the platform more dynamic and interactive. To do this, I thought it would be very useful to have a small service that works as a Pub/Sub system for clients. The idea was simple: when clients are interested in a topic (e.g., the comment or like counter on a given displayed post), they can subscribe to that topic (such as "number of likes" for comment ID 123). Through that subscription, they will get real-time updates when people are liking that post.

This idea originally came about during one of our bi-yearly hackathons, and one thing that was very important for me in the design of this architecture was a shared-nothing, scalable design.

![Real-time infrastructure overview](/substack-realtime-intro.png)

### **Build vs. Buy**

Before deciding to hand-roll our own implementation, we briefly looked at other implementations. They seemed to be designed for much smaller scales than we were looking for, and they required implementing frontend libraries that we didn't feel we needed in 2020. Most browsers that we cared about had [robust WebSocket implementations](https://caniuse.com/?search=websocket), and we wanted to use native libraries wherever possible, both on the back-end and then on the frontend.

For our backend, the technology of choice here was TypeScript/Node.js because that was our main programming language at Substack. For the Pub/Sub system, we used Redis Pub/Sub because Redis is plain the best :) It's one of the most robust pieces of technology I've ever used.

### Naive implementation

It's very easy to design a naive implementation of this whereby you have a frontend that allows clients to connect via web socket. Your frontend subscribes to that given topic in the Pub/Sub system and then pushes out those updates.

<iframe src="/diagrams/dia1.html" width="100%" height="450" frameborder="0" style="border-radius: 8px; margin: 15px 0;"></iframe>

**Problems with this Architecture:**
- **Single point of failure** - The entire Pub/Sub system can go down, taking all real-time functionality with it
- **Hot key problem** - When millions subscribe to the same topic, throughput is limited to that one box handling all Pub/Sub operations
- **All Frontend instances bottleneck** - Every frontend must connect to the same Pub/Sub instance, creating a scaling ceiling
- **No horizontal scaling** - Adding more frontends doesn't help when the Pub/Sub system becomes the bottleneck

### We shard

So the next step you could come up with is a sharded Pub/Sub system. In that case, the key space is divided into shards, and each frontend box is connected to the appropriate shard to push and receive updates from.

This is already much better than our first idea.

<iframe src="/diagrams/dia2.html" width="100%" height="450" frameborder="0" style="border-radius: 8px; margin: 15px 0;"></iframe>

**Improvements over Simple Architecture:**
- No single point of failure - shards can fail independently
- Horizontal scaling - add more shards as needed
- Load distribution across multiple Pub/Sub instances
- Each frontend only connects to relevant shards based on client subscriptions
- Key space partitioning enables better resource utilization


 An additional optimization we put in here is to differentiate between server and client subscriptions.

- A **client subscription** is as the name implies - the client is interested in a topic and wants to receive updates from it.
- A **server subscription** is the server subscribing to our Pub/Sub system for those given updates.

When a client subscribes to a topic that the servers are not yet subscribed to, the server subscribes to the Pub/Sub backend. For all subsequent clients connecting to that same frontend and subscribing to that same topic, the server simply handles the fan-out internally.

This way, we can handle the most difficult case where, for example, a million clients are subscribed to the same topic. If we have a maximum of 10,000 clients per frontend, we only have a hundred frontends subscribing to our Pub/Sub system. This is very feasible, because if we didn't have this optimization, our Pub/Sub system would have to deliver a separate message for each of the million clients.


One of the potential pitfalls of this design is that if one of the frontends dies, the clients connected to it will lose their real-time subscriptions. Thankfully, we're able to get away with storing the client subscriptions only on the frontend in memory due to a variety of reasons:

1. **The service was just incredibly stable.** It was a very simple Node.js, WebSocket, and Redis implementation where there were basically no bugs found in the years of runtime, despite massive usage.
2. **We over-provisioned the service** to handle sudden failures or traffic spikes.
3. When the client got a disconnect from the server, **it would automatically reconnect and re-subscribe** to its topics.
4. **Sessions were relatively short-lived.** When we scaled up the number of frontend instances during a traffic surge, clients naturally reconnected and spread themselves out across the new capacity.

### We scale

Zync turned out to be one of the most robust pieces of infrastructure I've ever written. There were only two notable commits in that repository:

1. The initialization commit.
2. A couple of months after rollout, one of our app developers asked for the ability to extend the API to allow subscribing and publishing messages to multiple topics at the same time. And so that API was expanded to take in either a string or a list of strings.

One of the biggest stress tests of the system happened during election night when, in a one-hour span, traffic went from **13 messages per second to 13,000 messages per second.** And the system handled it without any problems.

By the end of my tenure at Substack, Zync had grown to handle a **billion messages per week** and became a very important piece of infrastructure for us that was used throughout all of our clients, be they web or mobile.