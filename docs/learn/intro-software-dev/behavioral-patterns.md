---
slug: behavioral-patterns
title: "Behavioral patterns: observer, strategy, state"
description: Behavioral patterns govern how objects interact. Learn Observer for publish/subscribe events, Strategy for swappable algorithms, and State for behavior that changes with state.
keywords: behavioral patterns, observer pattern, strategy pattern, state pattern, publish subscribe, event, swappable algorithm, state machine, design patterns
level: intermediate
status: full
faq:
  - q: "What are behavioral patterns about?"
    a: "Behavioral patterns describe how objects *interact and divide responsibility* at runtime — the patterns of communication between objects. Rather than how objects are built (creational) or composed (structural), they focus on flow of control and notifications: who tells whom about a change, which algorithm runs, and how behavior shifts as conditions change."
  - q: "When should I use Strategy instead of an if/else?"
    a: "Use **Strategy** when you have several interchangeable algorithms for the same job and want to choose among them at runtime, add new ones easily, or test them in isolation. A small fixed `if`/`else` is fine for two stable choices. Strategy pays off when the set of algorithms grows, varies by configuration, or you want each one as a self-contained, swappable object."
  - q: "How is the State pattern different from a big switch statement?"
    a: "A switch on a status field scatters the per-state logic across every method that checks it. The **State** pattern puts each state's behavior in its own object and lets the current state object decide what happens and when to transition. Behavior changes by swapping the state object, so adding a new state means adding one class instead of editing every switch."
---
# Behavioral patterns: observer, strategy, state

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Observer** — objects subscribe to events and get notified when something changes, without the source knowing who is listening. **Strategy** — wrap interchangeable algorithms so you can swap them at runtime. **State** — let an object change its behavior by changing which state object is active.
</div>

Creational patterns make objects; structural patterns arrange them. **Behavioral patterns** handle the third question: once the objects exist and are connected, *how do they talk to each other and divide up the work?* These patterns are about communication and responsibility — who notifies whom, which algorithm runs, and how an object's behavior shifts as its situation changes. Three are essential: **Observer** for event notifications, **Strategy** for swappable algorithms, and **State** for behavior that depends on state. A trunking scanner uses all three, so the examples will stay close to home.

## What behavioral patterns are for

The hard part of a running program is rarely a single object — it is the *interactions*. When a call is decoded, the UI, the logger, and the recorder all need to know. When the user picks a demodulation mode, the right algorithm has to run. When a trunking follower moves from idle to active, its whole behavior changes. You *could* handle each with ad-hoc code, but that tends to scatter logic and tighten coupling. Behavioral patterns give these interactions a clean, named shape, mostly by keeping the parties from depending too directly on each other — another application of low [coupling](/learn/intro-software-dev/abstraction-coupling/).

## Observer

The **Observer** pattern defines a one-to-many dependency so that when one object (the *subject*) changes state, all its dependents (the *observers*) are notified automatically. It is the engine behind publish/subscribe, event listeners, and most UI frameworks.

The crucial property is decoupling: the subject does not know who is listening or what they will do. It just publishes events; anyone interested subscribes. In a scanner, the decode engine publishes a "new call decoded" event, and several unrelated parts react:

```text
class CallFeed {                 // the subject
  observers = []
  subscribe(o)   { observers.add(o) }
  publish(call)  { for o in observers: o.onCall(call) }
}

// independent observers, each does its own thing:
ui.subscribe       -> onCall(c): displayRow(c)
logger.subscribe   -> onCall(c): writeLine(c)
recorder.subscribe -> onCall(c): startRecording(c)

decodeEngine.feed.publish(newCall)   // everyone reacts
```

Add a new reaction — say, a webhook notifier — by writing one observer and subscribing it. The decode engine never changes. Remove one by unsubscribing. The trade-off to watch: with many observers and frequent events, notification order and cascading updates can get hard to reason about, and you must remember to unsubscribe to avoid leaks. But for "many things care when X happens," Observer is the standard answer.

There is a design choice inside Observer worth naming: **push** versus **pull**. In a *push* model the subject hands the observers the changed data directly (`onCall(call)` above), which is convenient when every observer needs the same payload. In a *pull* model the subject just signals "something changed" and each observer asks back for the details it wants. Push is simpler and common; pull is handy when different observers need different slices of the state and you do not want to over-share. The same pattern reappears across the stack at a larger scale — it is the foundation of event-driven architectures, which the [architectural patterns](/learn/intro-software-dev/architectural-patterns/) lesson returns to.

## Strategy

The **Strategy** pattern defines a family of interchangeable algorithms, encapsulates each one as an object behind a common interface, and makes them swappable at runtime. The client picks a strategy and uses it without knowing the algorithm's internals.

Radio work is full of interchangeable algorithms. Demodulation is the obvious one: FM, AM, and SSB are different math for the same job — turn baseband samples into audio. Rather than a giant switch buried in the pipeline, make each a strategy:

```text
interface Demodulator { demod(samples) -> audio }

class FmDemod  implements Demodulator { demod(s) { ...FM math... } }
class AmDemod  implements Demodulator { demod(s) { ...AM math... } }
class SsbDemod implements Demodulator { demod(s) { ...SSB math... } }

class Receiver {
  demod: Demodulator           // the current strategy
  setMode(d) { demod = d }     // swap at runtime
  process(s) { return demod.demod(s) }
}

rx.setMode(FmDemod())          // user flips a mode switch
```

The receiver holds a `Demodulator` and delegates to it. Switching modes is swapping the strategy object — no branching logic inside `process`. The same shape fits choosing a *filtering* strategy, a squelch algorithm, or a sort order in a totally different program. Strategy is the open/closed principle from [SOLID](/learn/intro-software-dev/solid/) made concrete: add a new algorithm by adding a class, never by editing the chooser.

Strategy and the structural Decorator can look similar, but the intent differs: Strategy *replaces* the algorithm for a step; Decorator *wraps* to add behavior around an existing one.

## State

The **State** pattern lets an object alter its behavior when its internal state changes — the object appears to change its class. You give each state its own object, move the state-specific behavior into those objects, and let the current state decide what to do and when to transition to another state.

A trunking *follower* is a textbook example. As it tracks a control channel, it moves through distinct modes, and what it does with an incoming message depends entirely on which mode it is in:

| State | Behavior | Transitions to |
|-------|----------|----------------|
| **Idle** | Listen to the control channel only | Affiliated, when a system is identified |
| **Affiliated** | Watch for grants on talkgroups of interest | Active, when a grant arrives |
| **Active** | Tune the voice channel and decode the call | Affiliated, when the call ends |

Without the pattern, every method becomes a tangle of `if state == ...` checks, and the same status variable is interpreted in a dozen places. With the State pattern, each mode is an object that knows its own behavior and its own exits:

```text
interface FollowerState { onMessage(ctx, msg) }

class Idle implements FollowerState {
  onMessage(ctx, msg) {
    if msg.identifiesSystem: ctx.setState(Affiliated())
  }
}
class Affiliated implements FollowerState {
  onMessage(ctx, msg) {
    if msg.isGrant && interested(msg): ctx.setState(Active(msg.channel))
  }
}
class Active implements FollowerState {
  onMessage(ctx, msg) {
    if msg.isCallEnd: ctx.setState(Affiliated())
    else: decodeVoice(msg)
  }
}

// the follower just delegates to its current state:
class Follower { state; onMessage(m) { state.onMessage(this, m) } }
```

Now behavior changes by swapping `state`, and adding a new mode (say, a *Hold* state) means adding one class, not editing every method. This is the **state machine** idea — a finite set of states with defined transitions — expressed as objects. The trade-off is more small classes, but for anything with genuinely distinct modes and rules about moving between them, that structure is far clearer than scattered conditionals.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Strategy encapsulates interchangeable algorithms so you can swap them at runtime." markdown="0">
  <p class="knowledge-check__q">Quick check: You want to let the user switch between FM, AM, and SSB demodulation at runtime. Which pattern fits?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Observer</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Strategy</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Singleton</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Behavioral patterns govern interaction** — how objects communicate and divide responsibility at runtime, mostly by keeping the parties loosely coupled.
- **Observer** — a subject notifies many observers when it changes, without knowing who is listening; the basis of publish/subscribe and UI events.
- **Strategy** — interchangeable algorithms behind one interface, swappable at runtime; pick a demodulator or filter without branching logic.
- **State** — each state is an object owning its behavior and transitions; a trunking follower moves through idle / affiliated / active by swapping the state object.
- **State machine** — the State pattern is the object-oriented expression of a finite state machine, far clearer than scattered status checks.
- **Open/closed in action** — Observer, Strategy, and State all let you add reactions, algorithms, or states by adding classes, not editing existing code.

Next up: scaling these ideas to real-time data — concurrency and streaming patterns, where pipelines, producers and consumers, and backpressure dominate signal software. See [Concurrency & streaming patterns](/learn/intro-software-dev/concurrency-and-pipelines/).
