---
title: "How to Build a modal.com Clone in 24 Hours"
date: "2025-11-30"
summary: "Or: How I Learned to Stop Worrying and Love the CRIU Dump. Building a serverless GPU platform with sub-second cold starts in a day of focused work."
description: "A practical guide to building a modal.com clone in 24 hours, covering SDK design, Kubernetes controllers, GPU snapshots with CRIU, and lazy container loading with Nydus."
readTime: true
autonumber: true
math: false
showTags: true
hideBackToTop: false
draft: true
tags: ["kubernetes", "gpu", "serverless", "infrastructure", "criu"]
---

Look, I've been doing this long enough to know that "build X in 24 hours" titles are usually clickbait. Someone hacks together a todo app, slaps enterprise terminology on it, and calls it a day. But what if I told you that you *can* build something genuinely useful—a serverless GPU platform with sub-second cold starts—in about a day of focused work?

Grab your energy drink of choice. Let's talk about building a modal.com clone.

## My Modal Journey (Or: Infrastructure as Code vs Code as Infrastructure)

I first encountered Modal at Topstack, where I was tech lead for the systems team. We were all-in on AWS. Everything was Infrastructure as Code—Terraform modules, CloudFormation templates, the whole nine yards. We had standardization. We had guardrails. We had order.

Then our head of data snuck Modal into the codebase.

Within weeks, half our recommendation pipeline was running on it—training scripts, batch inference jobs, the works. And my first reaction? Horror. Pure, unfiltered infrastructure-engineer horror.

There's no Infrastructure as Code with Modal. It's *Code as Infrastructure*. Resource names are implicit, derived from function and module names at runtime. You rename a function, you get a new deployment. No state file. No plan step. No import blocks. Just... Python decorators creating GPU clusters in the cloud.

Coming from AWS, it felt like chaos. Where were the CloudWatch dashboards? The VPC configurations? The IAM policies? Modal was fast and developer-friendly, sure, but it was missing so many features we'd come to expect from a "real" cloud provider.

So naturally, I did what any infrastructure engineer would do: I migrated our GPU inference workloads to Kubernetes. Proper deployments. Proper manifests. Proper infrastructure.

And *then*—only after running our recommendation pipeline on a beautifully orchestrated K8s cluster—did I actually start to appreciate what Modal had built.

Because here's the thing: to deliver that "never leave your code" developer experience, to make cold starts feel instant, they didn't just wrap Kubernetes in a nice API. They built a custom runtime filesystem. A custom scheduler. A custom everything. The seamlessness wasn't a thin veneer over existing tools—it was deep technical innovation at every layer.

That realization is what led me here. I wanted to understand how you actually build something like that. Not use it, not critique it, but *build* it. Turns out, you can get surprisingly far in 24 hours.

## What Even Is Modal?

If you haven't had the pleasure, [modal.com](https://modal.com) is basically black magic for GPU computing. You write Python with decorators, and somehow your code runs on A100s in the cloud. No Dockerfiles. No Kubernetes manifests. No crying into your keyboard at 2 AM wondering why your pod is `ImagePullBackOff` again.

```python
@app.function(gpu="A100")
def generate_image(prompt: str) -> bytes:
    pipe = load_sdxl()
    return pipe(prompt).images[0]
```

That's it. That's the DX we're chasing.

## The 24-Hour Breakdown

Here's the dirty secret: building a *working* version of this isn't actually that hard. Building a *production-ready* version would take months. But we're not here for production-ready—we're here for "this actually works and I understand why."

### Hour 0-4: The SDK Layer (The Fun Part)

Start with the developer experience, because if this sucks, nothing else matters.

The SDK is deceptively simple: an `Image` class for building containers, an `App` class for grouping functions, and a decorator that captures function metadata. Python's introspection capabilities do the heavy lifting:

```python
@dataclass
class Image:
    builder_version: str
    base: BaseImage
    steps: tuple[ImageStep, ...]

    @classmethod
    def debian_slim(cls, python_version="3.11"):
        return cls(base=BaseImage(ref=f"python:{python_version}-slim"))

    def pip_install(self, *packages):
        step = ImageStep(op="pip_install", args={"packages": list(packages)})
        return replace(self, steps=self.steps + (step,))
```

The chainable API (`Image.debian_slim().pip_install("torch").apt_install("git")`) is just immutable dataclasses returning new instances. No rocket science.

For the function decorator, you're capturing the callable, GPU requirements, and timeout. Then you generate a Kubernetes Job manifest on the fly when `.remote()` is called. Inspect the function source with `inspect.getsource()`, serialize arguments with JSON, wrap it in a runner script, and ship it.

Is this elegant? Not particularly. Does it work? Absolutely.

### Hour 4-10: The Controller (The Kubernetes Part)

Here's where things get real. You need something watching your cluster, managing GPU pods, and routing requests. This is the controller.

Three components:

**1. Pool Manager**: Tracks warm GPU pods. When a request comes in, it either finds a pod with the model already loaded, grabs an idle pod and loads the model, or evicts an LRU model from a busy pod. Classic cache management, but the cache entries are 80GB language models on $30/hour GPUs.

```python
async def acquire_pod(self, model_id: str, snapshot_url: str):
    # Priority 1: Already loaded (instant)
    for pod in self._pods.values():
        if pod.loaded_model == model_id:
            return pod

    # Priority 2: Idle pod (needs restore)
    for pod in self._pods.values():
        if pod.state == PodState.IDLE:
            await self._restore_model(pod, model_id, snapshot_url)
            return pod

    # Priority 3: Evict LRU
    evictable = sorted(pods, key=lambda p: p.last_used_at)
    ...
```

**2. Autoscaler**: Watches queue depth and utilization, adds pods when you're overwhelmed, removes them when idle. Cooldowns prevent thrashing. The algorithm is laughably simple—basically "if queue > 0, add pod; if idle for 5 minutes, remove pod"—but simple works.

**3. Router**: HTTP endpoint that accepts inference requests, acquires a pod from the pool, forwards the request, releases the pod. Streams responses for long-running inference.

I used Custom Resource Definitions to make this Kubernetes-native:

```yaml
apiVersion: gpufunctions.modal.dev/v1
kind: GpuPool
spec:
  replicas: 3
  gpu: "A100"
  image: "my-inference-image:latest"
```

### Hour 10-14: Image Building (The Nydus Part)

Traditional container images are slow. Pulling a 20GB PyTorch image takes forever, and you do it every time a pod starts.

Nydus (from the Dragonfly project) is lazy-loading for container images. Instead of downloading the entire image upfront, it fetches blocks on-demand. The base Python layer downloads immediately; the torch weights stream in as needed during model loading.

The image builder converts your chainable `Image` definition into actual layers:

```python
def build(self, spec: ImageSpec, tag: str):
    # Generate Dockerfile from spec
    dockerfile = self._generate_dockerfile(spec)

    # Build with Docker
    docker_build(dockerfile, tag)

    # Convert to Nydus format
    nydusify_convert(tag, nydus_tag)
```

This lazy-loading approach shaves precious seconds off cold starts. You don't wait for the full image—you start running as soon as you have the base layers.

### Hour 14-18: GPU Snapshots (The Magic)

This is the secret sauce. Cold starts on GPU workloads are *brutal*—loading an SDXL model can take 30+ seconds. Users will not wait 30 seconds.

Enter CRIU (Checkpoint/Restore In Userspace) with NVIDIA's cuda-checkpoint plugin. The concept:

1. Start a pod, load your model into GPU memory
2. Checkpoint the entire process state (CPU + GPU memory)
3. Store the checkpoint somewhere fast
4. When a new request comes in, restore from checkpoint instead of cold-starting

The implementation is surprisingly straightforward:

```python
async def restore_process(checkpoint_dir: Path) -> RestoreResult:
    # 1. CRIU restore (CPU state)
    proc = await asyncio.create_subprocess_exec(
        "criu", "restore",
        "--images-dir", str(checkpoint_dir),
        "--restore-detached",
        ...
    )

    # 2. CUDA restore (GPU memory)
    cuda_proc = await asyncio.create_subprocess_exec(
        "cuda-checkpoint",
        "--action", "restore",
        "--pid", str(restored_pid),
    )
```

The tricky bits:
- GPU snapshots are tied to specific hardware. An A100-40GB snapshot won't restore on an A100-80GB. You need affinity rules.
- Driver versions matter. NVIDIA 550 ≠ NVIDIA 545.
- The restored process picks up exactly where it left off, including open file handles and network sockets. Plan accordingly.

In benchmarks, this dropped SDXL cold starts from 27 seconds to 12 seconds—a 2.3x improvement. Not bad for what's essentially "just save and restore the process."

### Hour 18-24: The Runner (The Pod Runtime)

Each GPU pod runs a FastAPI server that speaks a simple protocol:

- `GET /health` - Am I alive?
- `GET /status` - What model do I have loaded?
- `POST /restore` - Load a model from snapshot
- `POST /infer` - Run inference
- `POST /evict` - Unload current model

The runner is the bridge between Kubernetes and your actual ML code. It manages the restored inference process, proxies requests to it, and handles cleanup on eviction.

One design decision that saved headaches: the restored ML process runs on a separate port (8001) from the runner (8000). The runner proxies requests and handles lifecycle. This isolation means a crashed inference process doesn't take down the pod—you can just restore again.

---

And that's the core system. Nydus gets your images loaded fast. CRIU restores your GPU state instantly. The runner coordinates everything. The controller orchestrates the chaos. Combined, you get cold starts that feel warm.

## The Numbers That Matter

After all this work, what did we actually achieve? Here are the benchmarks from running SDXL on a V100:

| Metric | Traditional Cold Start | With GPU Snapshot |
|--------|----------------------|-------------------|
| Time to first inference | 27 seconds | 12 seconds |
| **Speedup** | baseline | **2.3x faster** |

That's not theoretical—that's measured. The breakdown: about 3 seconds for Nydus image streaming, 4 seconds for CRIU process restore, 5 seconds for CUDA memory restore. Compare that to: pull image (8s), start container (2s), import torch (4s), load model weights (13s).

The real win is on subsequent requests with the same model. Pod already warm? We're talking milliseconds, not seconds. That's the difference between "usable" and "feels instant."

## What's Missing (aka The Other 6 Months)

Let's be honest about what we skipped:

- **Authentication/multi-tenancy**: Who can run what? Resource quotas? Billing?
- **Secrets management**: How do models get HuggingFace tokens?
- **Persistent storage**: Volumes, model caches, training checkpoints
- **Networking**: Private endpoints, VPCs, ingress
- **Observability**: Metrics, tracing, log aggregation
- **The literal entire frontend**: Dashboard, deployment history, spend tracking

Building infrastructure is an iceberg. The technical core—scheduler, snapshots, SDK—is the visible tip. Everything else is underwater, silently keeping users from capsizing.

## Lessons Learned

**1. Start with DX, work backward.** We wrote `@app.function(gpu="A100")` before we had a single line of Kubernetes code. Knowing what we wanted to type clarified what we needed to build.

**2. CRIU is underrated.** The checkpoint/restore ecosystem is mature and well-documented. GPU support is newer but works. Don't be afraid of process-level snapshotting.

**3. Kubernetes is your friend (eventually).** Yes, the learning curve is vertical. But CRDs + controllers + operators give you a framework for building platform infrastructure. Fight it less, leverage it more.

**4. Simple scheduling beats clever scheduling.** Our autoscaler is dumb. Add pods when busy, remove pods when idle. That's it. I've seen teams spend months on "optimal bin packing" only to have simpler systems outperform them in practice.

**5. The demo is the spec.** Write example code before implementation code. `sdxl_inference.py` existed before most of the platform did. It kept us honest about what developers actually need.

## Should You Build This?

Probably not? Modal, Replicate, and RunPod exist. They have teams, funding, and production battle-scars.

But if you're trying to *understand* how these platforms work—if you want to peek behind the curtain—building a clone is incredibly educational. You'll learn more about containers, GPUs, scheduling, and process isolation in 24 hours than in months of reading documentation.

And hey, maybe your clone turns into something real. Stranger things have happened.

Now if you'll excuse me, I have some CUDA contexts to checkpoint.

---

*The full source code is available at [wherever you're reading this]. PRs welcome, especially if they involve fixing my questionable use of asyncio.*
