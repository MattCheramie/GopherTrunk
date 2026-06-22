---
slug: tablets
title: Tablets
description: A tablet is a big-screen touch computer sitting between phone and laptop. Learn where the extra size pays off, what runs on tablets, and their limits as a primary dev machine.
keywords: what is a tablet, tablet computer, iPad Pro, iPadOS, Android tablet, tablet vs laptop, tablet vs phone, kiosk hardware, point of sale tablet, tablet for development
level: beginner
status: full
faq:
  - q: "How is a tablet different from a phone?"
    a: "Mostly size. A tablet runs the same kind of operating system as a phone — **iPadOS** or **Android** — on a much larger touch screen. That bigger canvas changes what it's good at: reading, drawing, side-by-side apps, and kiosk displays all benefit from the room. High-end tablets like the iPad Pro even use laptop-class **M-series** chips, blurring the line further."
  - q: "Can a tablet replace my laptop?"
    a: "For some people, for some tasks — but rarely fully. Tablets handle media, reading, note-taking, and light productivity well, and an iPad can now run a few genuine development tools. But they remain fairly **locked down**, lean on touch over keyboard and mouse, and lack the freedom of a real desktop OS. As a primary development machine, a tablet is a compromise."
  - q: "What languages do you use to build tablet apps?"
    a: "The same ones as phones. iPad apps are built in **Swift** with Xcode; Android tablet apps in **Kotlin** with Android Studio; and cross-platform frameworks like **Flutter** and **React Native** cover both. A well-built mobile app usually adapts its layout to the larger screen rather than being a separate product."
---
# Tablets

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A tablet is essentially a large-screen phone** — same iPadOS or Android lineage, sometimes laptop-class silicon like the iPad Pro's M-series chips. **The extra size pays off** for media, reading, drawing, kiosks, point-of-sale, and field tools where a phone is too small and a laptop too bulky. **It sits between phone and laptop** — more capable than one, less open than the other, and a compromise as a primary dev machine.
</div>

A tablet is the phone's big-screened cousin. Architecturally it's almost the same animal — an ARM-based touch computer running a mobile operating system — but the larger display shifts what it's good for. This lesson looks at where that extra size earns its keep, what runs on tablets, and why "between phone and laptop" cuts both ways.

## What a tablet is

At its core, a tablet is a large phone. It runs the same family of operating systems — **iPadOS** on Apple's iPads, **Android** on most others — built around touch, apps, and a battery, with no built-in keyboard. Everything you learned about phone hardware applies: ARM SoC, sensors, radios, sealed design.

The interesting wrinkle is at the high end. Apple's iPad Pro and iPad Air now run the same **M-series** chips found in MacBooks — genuinely laptop-class processors. That gives a top-tier tablet the raw horsepower of a real computer, even while the software around it stays closer to a phone than a desktop. So a tablet can range from a cheap media-consumption slab to a machine more powerful than many laptops, all in the same form factor.

## Where the extra size pays off

A bigger screen isn't a minor upgrade — it unlocks whole categories of use that a phone handles poorly:

- **Media and reading.** Movies, magazines, comics, and documents are far more comfortable on a tablet-sized display.
- **Drawing and creative work.** With a stylus, tablets are first-class tools for illustration, sketching, photo editing, and note-taking.
- **Point-of-sale and kiosks.** The flat, durable, single-app tablet is the default hardware for cash registers, check-in screens, menus, and self-service kiosks.
- **Field tools.** Inspectors, clinicians, and technicians use tablets for forms, diagrams, and reference material where a phone is cramped and a laptop is awkward to hold.
- **Light productivity.** Email, browsing, documents, and video calls all work well, especially with a clip-on keyboard.

This is also where a tablet earns a place in a setup like GopherTrunk's: propped on a desk, a tablet makes a roomy, glanceable dashboard for a scanner running on a server elsewhere — more screen than a phone, less commitment than a laptop.

## Phone vs tablet vs laptop

The clearest way to place a tablet is between its neighbors. Each device wins at something the others don't:

| Trait | Phone | Tablet | Laptop |
|-------|-------|--------|--------|
| **Screen size** | Small, pocketable | Large touch screen | Large, plus keyboard |
| **Portability** | Always with you | Carried, not pocketed | Bag-sized |
| **Primary input** | Touch | Touch (+ stylus / keyboard) | Keyboard & trackpad |
| **Battery life** | All day | Often very long | Half to full day |
| **Openness** | Locked down | Fairly locked down | Fully open OS |
| **Best at** | On-the-go, sensors | Media, drawing, kiosks | Real work & development |
| **Dev machine?** | No | Limited | Yes |

The pattern is plain: the phone wins on portability and sensors, the laptop wins on openness and real work, and the tablet sits in the middle — bigger and nicer to look at than a phone, but more constrained than a laptop. For choosing the laptop end of that spectrum deliberately, see [Choosing a development machine](/learn/intro-hardware/choosing-a-dev-machine/).

## Strengths

Tablets do several things better than anything else:

- **Big, comfortable touch screen.** The best surface for reading, watching, drawing, and showing things to other people.
- **Portable and quiet.** Lighter than a laptop, instant-on, fanless, and easy to hand around or mount on a wall.
- **Long battery life.** Mobile-class efficiency plus a large body usually means excellent endurance.
- **Great for content and kiosks.** Durable, lockable to a single app, and cheap enough to deploy in numbers — ideal for displays, registers, and field use.

If the job is consuming, presenting, drawing, or showing, a tablet is often the most pleasant tool available.

## Drawbacks

The same in-between position creates real limits:

- **Jack of both, master of neither.** A tablet is less portable than a phone and less capable than a laptop, so for many people it's a third device rather than a replacement for either.
- **Still fairly locked down.** Like phones, tablets run a sealed mobile OS with store-gated apps and limited access to the system. iPadOS has opened up somewhat, but it's not a free desktop.
- **Limited as a primary dev machine.** Even with laptop-class chips, the software environment, touch-first input, and restricted tooling make a tablet a compromise for serious development. An iPad can now run *some* real dev tools, but most coding still happens on a desktop or laptop.

A tablet is a wonderful companion device and an excellent appliance. It just rarely wants to be the one machine you do everything on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a tablet sits between phone and laptop: bigger and nicer than a phone, but more locked down and less capable than a laptop." markdown="0">
  <p class="knowledge-check__q">Quick check: Where does a tablet sit relative to a phone and a laptop?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's identical to a laptop, just without a keyboard</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Between them — bigger and more comfortable than a phone, but more locked down and limited than a laptop</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's weaker than a phone in every way</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A tablet is a large phone** — same iPadOS or Android lineage, sometimes with laptop-class M-series chips.
- **Size unlocks new uses** — media, reading, drawing, kiosks, point-of-sale, and field tools all benefit from the room.
- **It sits between phone and laptop** — more comfortable than one, more open than the other, master of neither.
- **Same app stacks as phones** — Swift, Kotlin, and cross-platform frameworks, with layouts that adapt to the bigger screen.
- **A compromise as a dev machine** — still fairly locked down and touch-first, so serious development stays on a real computer.

Next up: the three ways to actually build a mobile app, the toolchains behind each, and how to choose. See [Developing for mobile devices](/learn/intro-hardware/developing-for-mobile/).
