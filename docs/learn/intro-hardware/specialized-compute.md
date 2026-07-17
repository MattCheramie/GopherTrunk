---
slug: specialized-compute
title: Beyond the CPU — GPUs, FPGAs & accelerators
description: The CPU can run any code, but it isn't always the best tool. Learn how GPUs, FPGAs, DSPs, and AI accelerators trade flexibility for throughput and efficiency — and when SDR work actually needs them.
keywords: GPU, graphics processing unit, FPGA, field programmable gate array, DSP chip, ASIC, AI accelerator, NPU, TPU, parallel computing, SDR FPGA
level: intermediate
status: full
prereq:
  - building-blocks
faq:
  - q: "What's the difference between a CPU and a GPU?"
    a: "A **CPU** has a few powerful cores optimised for running any kind of code fast, including branchy, decision-heavy logic. A **GPU** has thousands of small, simple cores that all do the *same* operation on different data at once. The CPU is the versatile generalist; the GPU is a specialist that wins big on wide, repetitive math — image processing, physics, and the matrix multiplications behind modern AI — but is poor at serial, branchy work."
  - q: "What is an FPGA?"
    a: "An **FPGA** (field-programmable gate array) is a chip you can rewire into a custom digital circuit after it's manufactured. Instead of running software on a fixed processor, you describe hardware — filters, counters, decoders — and the FPGA becomes that circuit. The payoff is deterministic, low-latency, massively parallel processing, which is why high-end SDRs use one to filter and decimate samples right at the antenna before a PC ever sees them. The cost is that FPGAs are much harder to program than ordinary computers."
  - q: "Do I need a GPU for SDR?"
    a: "Almost certainly not. Most SDR decoding, GopherTrunk included, runs entirely on the CPU and decodes trunked voice comfortably on an ordinary laptop or even a Raspberry Pi. GPUs and FPGAs matter only at the high end — very wide bandwidths, many channels at once, or heavy real-time DSP research. A well-written CPU pipeline covers the everyday case."
  - q: "What is an AI accelerator or NPU?"
    a: "An **AI accelerator** is silicon built specifically for the math machine-learning models use — mostly large matrix multiplications. An **NPU** (neural processing unit) is one built into a phone or laptop chip; a **TPU** (tensor processing unit) is Google's data-centre version. They do far less than a general CPU but run neural-network math faster and with much lower power, which is why they're increasingly baked into everyday devices."
gophertrunk_links:
  - title: SDR hardware
    url: /hardware.html
    note: the receivers GopherTrunk runs on — some do their first-stage DSP on an onboard FPGA.
  - title: Architecture
    url: /architecture.html
    note: how GopherTrunk organises its DSP pipeline — and why it stays on the CPU.
---

# Beyond the CPU — GPUs, FPGAs & accelerators

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The CPU is a generalist** — a few fast cores that run *any* code, but not always the best tool for the job. When work is massively parallel or has to happen with hard timing guarantees, specialised silicon wins. A **GPU** throws thousands of small cores at wide, repetitive math (and powers modern AI). An **FPGA** becomes a custom circuit for deterministic, low-latency processing. A fixed-function **accelerator** (ASIC, NPU, TPU) trades away flexibility for maximum throughput and efficiency. The rule of thumb: reach past the CPU only when the CPU can't keep up. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/).
</div>

The [last lesson](/learn/intro-hardware/building-blocks/) opened the box and found four parts in every computer: a CPU, memory, storage, and I/O. That CPU is a general-purpose processor — it will run anything you ask of it. But "will run anything" and "is the best tool for this particular job" are not the same claim. Some jobs are shaped so awkwardly for a CPU that engineers reach for a different kind of chip entirely. This lesson names those chips, explains what each is good at, and — honestly — where they do and don't matter for SDR work.

## The CPU is a generalist

A CPU has a handful of powerful cores, each running its fetch-execute loop very fast. Its great strength is *versatility*: it handles branchy, decision-heavy code — "if this, do that; otherwise loop back" — as easily as straight-line arithmetic. Almost all software you'll ever write runs happily on a CPU, and for most tasks nothing else is needed.

Where a CPU struggles is at the extremes. Two in particular:

- **Massively parallel math.** A CPU with eight cores can do eight things at once. If your job is the *same* simple operation applied to a million pieces of data, eight-at-a-time is slow.
- **Hard real-time signal work.** A general-purpose operating system can pause your program at any moment to do something else. That's fine for a spreadsheet, but if a filter must produce a result every fixed number of nanoseconds with no exceptions, the CPU's flexibility becomes a liability.

Each of the chips below exists because it answers one of those two weaknesses.

## GPUs — thousands of small cores

A **GPU** (graphics processing unit) attacks the first weakness. Where a CPU has a few big cores, a GPU has *thousands* of small, simple ones. They aren't independent the way CPU cores are — they mostly march in lockstep, every core running the same instruction on a different slice of data. This is called **data-parallel** or SIMD (single instruction, multiple data) work.

That design was born for graphics: shading millions of pixels is exactly "do the same math to a huge grid of data." Engineers soon realised the same engine could crunch *any* wide parallel math — a use called **GPGPU** (general-purpose computing on a GPU). Today that's the engine behind modern AI: training and running neural networks is mostly enormous matrix multiplication, which maps perfectly onto thousands of cores.

The trade-off is the mirror image of the CPU's. Give a GPU branchy, serial logic — lots of "if this then that" where each step depends on the last — and most of its cores sit idle. GPUs also have their own dedicated memory, **VRAM**, and a model or dataset that won't fit in VRAM simply can't run well. A GPU is a specialist: spectacular on the right shape of problem, mediocre on the wrong one.

## FPGAs — reconfigurable hardware

An **FPGA** (field-programmable gate array) attacks the second weakness, and it does so in a fundamentally different way. A CPU and a GPU are fixed chips that *run software*. An FPGA is a chip full of logic blocks and wires you can **reconfigure into a custom circuit** after it leaves the factory. You don't write a program that runs on a processor — you describe hardware, and the FPGA physically *becomes* that hardware.

Because the result is a real circuit rather than software sharing a busy processor, an FPGA delivers **deterministic, low-latency, parallel** processing: every clock tick, the circuit does exactly what it was wired to do, on time, every time. That's precisely what high-rate signal processing wants. High-end SDRs put an FPGA right behind the antenna to **filter and decimate** the raw sample stream — throwing away the bandwidth you don't care about — before the data ever reaches a PC over USB or Ethernet. Doing that first stage in dedicated hardware is often the only way to keep up with a very wide, very fast stream.

The catch is programming. You describe an FPGA's circuit in a **hardware description language** (HDL) such as VHDL or Verilog, thinking in terms of gates, clocks, and timing rather than ordinary lines of code. It's a specialised skill, and development is slower and less forgiving than writing software.

## DSPs — chips built for signal math

A **DSP** (digital signal processor) sits between the general CPU and the fully custom FPGA. It's a processor — it runs software — but its instruction set and hardware are tuned for the one operation signal processing does constantly: **multiply-accumulate**, multiplying pairs of numbers and adding up the results, the heart of every digital filter.

DSPs powered generations of radios, modems, and audio gear, and they haven't gone away — they're still tucked inside phones, hearing aids, and countless embedded devices, quietly doing audio and radio math efficiently. On a modern PC the CPU is usually fast enough that a separate DSP chip isn't needed, but the *idea* — hardware shaped around signal math — lives on in CPU vector instructions and in FPGA and GPU pipelines alike.

## ASICs, NPUs & AI accelerators

At the far end of the spectrum is the **ASIC** (application-specific integrated circuit): a chip designed and manufactured to do exactly one thing, with nothing reconfigurable about it. Because every transistor is committed to the task, an ASIC is the most efficient and fastest option there is — and the least flexible. If the job ever changes, you need a new chip. ASICs make sense only when a job is fixed and the volume is huge: think the radio baseband chip in a phone, or a Bitcoin miner.

Modern **AI accelerators** are ASICs (or near-ASICs) built for machine-learning math. An **NPU** (neural processing unit) is one integrated into a phone or laptop processor; a **TPU** (tensor processing unit) is Google's data-centre design. They can't do much beyond neural-network math, but they do it far faster and at far lower power than a general CPU — which is why they're increasingly standard silicon.

Step back and there's a clean spectrum:

| | CPU | GPU | FPGA | ASIC |
|---|-----|-----|------|------|
| **Flexibility** | anything | wide parallel math | any circuit you wire | one fixed job |
| **Efficiency** | lowest | high on the right shape | high | highest |
| **Programmed with** | ordinary code | GPU code | hardware description | not programmable |

Read left to right and you're trading **flexibility for efficiency**. The CPU does anything but wrings the least throughput per watt from silicon; the ASIC does exactly one thing at maximum efficiency and nothing else. GPUs and FPGAs sit in between, each leaning one way.

## Where SDR & DSP work leans on these

Here's the honest framing, because it's easy to assume radio work demands exotic hardware. It usually doesn't.

At the **high end** — capturing hundreds of megahertz of spectrum, monitoring dozens of channels at once, or doing heavy real-time DSP research — you do see specialised compute. The SDR itself may carry an **FPGA** to filter and decimate before the data leaves the device, and a **GPU** can accelerate wide, batch-style DSP on the host.

But for the everyday task this path is about — decoding a trunked voice system — a **well-written CPU pipeline is plenty**. GopherTrunk's entire DSP chain, from down-conversion to demodulation to symbol decoding, runs on the CPU and keeps up comfortably on an ordinary laptop, and even on a [Raspberry Pi](/learn/intro-hardware/programming-an-sbc/) sitting at the antenna. The reason is that the receiver normalises the signal down to a modest per-channel rate early on, so the heavy math happens on a manageable stream — no GPU or custom silicon required. That's a deliberate design choice, and it's the opposite end of the spectrum from a [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) too small to run a general OS at all.

## Choosing

The heuristic is short: **reach for specialised compute only when the CPU genuinely can't keep up.** Specialised silicon is faster or more efficient on its niche, but it costs more, often draws its own power budget, and — for FPGAs and ASICs especially — takes far more effort to develop for.

So weigh the same three forces every hardware decision comes down to: [cost, power, and performance](/learn/intro-hardware/cost-power-performance/), plus the development effort a harder-to-program chip demands. If a CPU meets your needs, use it — it's the cheapest to build for and the easiest to change. Move to a GPU, FPGA, or accelerator when a real bottleneck forces the trade, not before.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an FPGA becomes a custom circuit, giving the deterministic, low-latency processing a filter at the antenna needs." markdown="0">
  <p class="knowledge-check__q">Quick check: you need a deterministic, low-latency filter running right at the antenna before samples ever hit the PC. Which chip is the best fit?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A faster general-purpose CPU</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">An FPGA</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A bigger GPU with more VRAM</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The CPU is a versatile generalist** — a few fast cores that run any code well, but struggle with massively parallel math and hard real-time timing.
- **GPUs** bring thousands of small cores for wide, data-parallel math — graphics, general compute, and the engine of modern AI — but are poor at branchy serial logic.
- **FPGAs** are reconfigurable hardware: you wire them into a custom circuit for deterministic, low-latency processing, at the cost of harder (HDL) development.
- **DSPs and ASICs/NPUs/TPUs** fill out a spectrum from CPU to fixed-function silicon that trades flexibility for efficiency.
- **SDR reality check** — the high end may use an FPGA or GPU, but a well-written CPU pipeline like GopherTrunk's decodes trunked voice on a plain laptop or a Raspberry Pi.
- **Choosing** — reach past the CPU only when it can't keep up, weighing cost, power, and development effort.

Next up: with every kind of processor named, we can lay all the platforms on one map — [the hardware spectrum, cloud to microcontroller](/learn/intro-hardware/hardware-spectrum/).
