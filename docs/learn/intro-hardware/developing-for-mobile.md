---
slug: developing-for-mobile
title: Developing for mobile devices
description: There are three ways to build a mobile app — native, cross-platform, and web. Learn the languages and toolchains each uses, the trade-offs, and why you build on a desktop and deploy to the phone.
keywords: mobile app development, native app development, cross-platform mobile, Flutter, React Native, .NET MAUI, progressive web app, PWA, Swift Xcode, Kotlin Android Studio, deploy to phone
level: intermediate
status: full
faq:
  - q: "What are the three main ways to build a mobile app?"
    a: "**Native** — written in each platform's own language and tools (Swift/Xcode for iOS, Kotlin/Android Studio for Android) for the best performance and platform feel. **Cross-platform** — one codebase that targets both stores, using Flutter (Dart), React Native (JavaScript), or .NET MAUI. **Web/PWA** — an ordinary web app built with HTML, CSS, and JavaScript that runs in the browser and can be installed to the home screen. Each trades polish, effort, and reach differently."
  - q: "Do I develop the app on the phone itself?"
    a: "No. You write, build, and test mobile apps on a **desktop or laptop** using the platform's toolchain, then deploy the finished app to the phone or to an emulator. iOS development in particular requires a Mac running Xcode. This is exactly why choosing a good development machine matters — the phone is the target, not the workshop."
  - q: "When should I choose a web app or PWA over a native app?"
    a: "Choose **web/PWA** when reach and zero-install matter more than deep device access — anyone with a browser can use it instantly, with no app store. Choose **native or cross-platform** when you need the best performance, smooth animations, or full access to sensors and platform features. Many products start as a web app for reach and add a native app later."
---
# Developing for mobile devices

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Three approaches** — native (Swift/Kotlin), cross-platform (Flutter, React Native, .NET MAUI), and web/PWA (HTML/CSS/JS). **The trade-off is performance and platform feel vs. one shared codebase vs. instant reach with no install.** **You build on a desktop and deploy to the phone** — the device is the target, the development happens on a real computer, often through an app store.
</div>

You've seen that phones and tablets are deployment targets, not workstations. This lesson is about the deployment itself: the concrete ways to turn an idea into an installable mobile app. There are three broad routes, each with its own languages, toolchains, and trade-offs — and one rule that applies to all of them: you do the work on a desktop or laptop and ship the result to the device.

## Approach 1: native

A **native** app is written in the language and tools that a platform was designed around. You build twice — once per platform — but you get the deepest access and the most authentic feel.

- **iOS** — written in **Swift** (older code in Objective-C) using **Xcode** on a Mac. Xcode handles building, the simulator, signing, and submission to the App Store.
- **Android** — written in **Kotlin** (or Java) using **Android Studio**, which runs on Windows, macOS, or Linux and includes an emulator and build tools.

Native gives you the best performance, the smoothest animations, and immediate access to every new platform feature and sensor the day it ships. The cost is duplication: two codebases, two languages, and two teams' worth of effort if you want both platforms.

## Approach 2: cross-platform

A **cross-platform** framework lets you write one codebase that runs on both iOS and Android. You trade a little native fidelity for a large saving in effort.

- **Flutter** — Google's framework, written in **Dart**. It draws its own UI, giving consistent results across platforms.
- **React Native** — written in **JavaScript/TypeScript**, popular with web developers because it reuses React concepts.
- **.NET MAUI** — Microsoft's framework, using **C#**, attractive for teams already in the .NET world.

These frameworks still build native app packages under the hood and can reach down to platform features when needed, often through plugins. The trade-off is that the very newest platform capabilities may lag, and squeezing out the last bit of native polish can take extra work. For most apps, the single shared codebase is well worth it.

## Approach 3: web and PWA

The third route skips the app stores almost entirely. A **web app** is built with the standard web stack — **HTML, CSS, and JavaScript** — and runs in the phone's browser. A **Progressive Web App (PWA)** is a web app enhanced so it can be installed to the home screen, work offline, and feel app-like, while still being just a website.

The appeal is reach with no friction: anyone with a browser can use it instantly, on any platform, with no download and no store review. The limits are real too — web apps have more restricted access to device sensors and platform features, and can't always match native performance or look-and-feel. But for content, dashboards, and tools that don't need deep hardware access, web is often the fastest way to everyone at once.

This is also the lightest way to put a screen on server-hosted software. A GopherTrunk dashboard served from the machine running the scanner can simply be opened in a phone or tablet browser — the device becomes a thin client, with all the real work staying on the server.

## Comparing the three approaches

| | Native | Cross-platform | Web / PWA |
|---|--------|----------------|-----------|
| **Languages** | Swift, Kotlin | Dart, JS/TS, C# | HTML, CSS, JS |
| **Codebases** | One per platform | One shared | One shared |
| **Performance & feel** | Best | Very good | Good enough for many apps |
| **Device & sensor access** | Full | Most (via plugins) | Limited |
| **Distribution** | App stores | App stores | A URL — no install needed |
| **Best when** | Polish and depth matter | Reaching both stores efficiently | Reach and zero-install matter |

There's no universally correct choice — it's the same "match the tool to the constraints" thinking the whole path teaches. Need maximum polish and a single platform? Go native. Want both stores from one team? Go cross-platform. Need everyone, instantly, with no install? Go web.

## You build on a desktop, not on the phone

Whatever route you pick, the development happens on a real computer. Xcode needs a Mac; Android Studio, Flutter, and the web toolchains run on a proper desktop or laptop with a keyboard, a big screen, and the freedom to install whatever you need. You write code, build it, run it in an emulator or on a tethered device, and only then deploy.

This is precisely why earlier lessons treated the phone as a target and stressed picking the right *development* machine — see [Choosing a development machine](/learn/intro-hardware/choosing-a-dev-machine/). The phone is where your software lives; the laptop is where it's made.

## App store realities

Distribution shapes mobile development as much as code does. Native and cross-platform apps generally ship through the **App Store** and **Google Play**, which means:

- **Review and approval.** Apple and Google inspect submissions; an app can be delayed or rejected.
- **Fees and revenue cuts.** Store accounts cost money, and the stores take a percentage of in-app sales.
- **Signing and identity.** Apps must be cryptographically signed and tied to a developer account.

Web apps and PWAs sidestep most of this — there's no review and no cut, just a URL — which is part of why "ship it on the web" is such a common starting point. Knowing the distribution rules before you start often steers the technical choice as much as performance does.

<div class="knowledge-check" data-quiz data-correct-msg="Right — native, cross-platform, and web/PWA are the three routes, trading platform feel, shared effort, and reach." markdown="0">
  <p class="knowledge-check__q">Quick check: What are the three broad approaches to building a mobile app?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Compiled, interpreted, and scripted</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Native, cross-platform, and web/PWA</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">iOS-only, Android-only, and Windows-only</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Native** — Swift/Xcode for iOS and Kotlin/Android Studio for Android; best performance and feel, but a codebase per platform.
- **Cross-platform** — Flutter (Dart), React Native (JS), or .NET MAUI (C#); one shared codebase across both stores.
- **Web / PWA** — HTML, CSS, and JavaScript in the browser, installable to the home screen; maximum reach, no install, but limited device access.
- **The trade-off** — polish and depth vs. shared effort vs. reach; choose by which constraint matters most.
- **Build on a desktop, deploy to the phone** — development happens on a real machine, and the device is the target.
- **App stores set rules** — review, fees, and signing shape native distribution, while web sidesteps all of it.

Next up: we leave handheld devices for the world of tiny, cheap, full computers you can build into anything. See [What is a single-board computer?](/learn/intro-hardware/what-is-an-sbc/).
