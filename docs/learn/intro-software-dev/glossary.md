---
slug: glossary
title: Glossary of software development terms
description: Plain-language definitions for every key software development term in the Intro to Software Development learning path, cross-linked to the lessons that explain them.
keywords: software development glossary, programming terms, programming glossary, software engineering definitions
level: beginner
status: full
lesson_standalone: true
---

# Glossary of software development terms

This is a plain-language reference for the key terms used across the Intro to Software Development path. Definitions are short on purpose; each links to the lesson that explains the idea in full. Terms are grouped by theme and roughly ordered from foundational to advanced within each group, following the shape of the path itself.

> **Want the long version?** Many of these terms — every programming language and
> core concept here — have a full, in-depth article in the
> [GopherTrunk Field Guide](/reference/), and the first mention of a covered term on
> any page links straight to it.

## History and hardware

**Hardware** — The physical parts of a computer you can touch: chips, memory, drives, screens. Software is the instructions that run on it. See [What is software?](/learn/intro-software-dev/what-is-software/)

**Software** — Instructions that tell hardware what to do. Unlike hardware, it can be changed without rebuilding the machine. See [What is software?](/learn/intro-software-dev/what-is-software/)

**Stored-program computer** — A design where a program is held in the same memory as the data it works on, so the machine can be reprogrammed instead of rewired. This idea underpins essentially every modern computer. See [What is software?](/learn/intro-software-dev/what-is-software/)

**Firmware** — Low-level software baked into a device that controls its hardware directly, sitting between pure hardware and ordinary programs. See [What is software?](/learn/intro-software-dev/what-is-software/)

**Operating system (OS)** — The software that manages a computer's hardware and resources and gives other programs a consistent way to run. See [What is software?](/learn/intro-software-dev/what-is-software/)

**Vacuum tube** — An early electronic switch used in the first electronic computers; large, hot, and unreliable compared with what replaced it. See [The story of computing hardware](/learn/intro-software-dev/computing-hardware-story/)

**Punch card** — A stiff card with holes punched in it, used to feed programs and data into early computers. See [The story of computing hardware](/learn/intro-software-dev/computing-hardware-story/)

**ENIAC** — One of the first general-purpose electronic computers (1940s), built from thousands of vacuum tubes. See [The story of computing hardware](/learn/intro-software-dev/computing-hardware-story/)

**Transistor** — A tiny electronic switch that replaced the vacuum tube, making computers far smaller, faster, and more reliable. See [The story of computing hardware](/learn/intro-software-dev/computing-hardware-story/)

**Integrated circuit** — Many transistors fabricated together on a single chip, the building block of modern electronics. See [The story of computing hardware](/learn/intro-software-dev/computing-hardware-story/)

**Microprocessor** — A complete processing unit on a single chip; the "brain" that runs a computer's instructions. See [The story of computing hardware](/learn/intro-software-dev/computing-hardware-story/)

**Moore's Law** — The observation that the number of transistors on a chip roughly doubles every couple of years, driving decades of rapid improvement in computing. See [The story of computing hardware](/learn/intro-software-dev/computing-hardware-story/)

**Mainframe** — A large, powerful central computer shared by many users, common in early business and institutional computing. See [From the internet to the edge](/learn/intro-software-dev/internet-to-edge/)

**Client-server** — A model where client programs request services from a central server that holds shared data or logic. See [From the internet to the edge](/learn/intro-software-dev/internet-to-edge/)

**Cloud** — Computing power and storage provided over the internet on demand, instead of from machines you own and run yourself. See [From the internet to the edge](/learn/intro-software-dev/internet-to-edge/)

**Edge computing** — Running computation close to where data is produced (devices, sensors, local servers) rather than in a distant data centre, to reduce latency. See [From the internet to the edge](/learn/intro-software-dev/internet-to-edge/)

**Open source** — Software whose source code is published so anyone can read, use, and contribute to it, usually under a licence that permits this. See [From the internet to the edge](/learn/intro-software-dev/internet-to-edge/)

## Languages and paradigms

**Machine code** — The raw numeric instructions a processor executes directly. Everything else eventually becomes machine code. See [The birth of languages](/learn/intro-software-dev/birth-of-languages/)

**Assembly** — A human-readable shorthand for machine code, with one line roughly per processor instruction. See [The birth of languages](/learn/intro-software-dev/birth-of-languages/)

**High-level language** — A language written in terms closer to human ideas than to the hardware, then translated down to machine code. See [The birth of languages](/learn/intro-software-dev/birth-of-languages/)

**Compiler** — A program that translates source code into another form, typically machine code or bytecode, before it runs. See [The birth of languages](/learn/intro-software-dev/birth-of-languages/)

**Paradigm** — A broad style of organising programs and thinking about problems, such as object-oriented or functional. See [Language families](/learn/intro-software-dev/language-families/)

**Imperative** — A style where you write step-by-step instructions that change the program's state. See [Language families](/learn/intro-software-dev/language-families/)

**Procedural** — An imperative style organised around procedures or functions that you call to do work. See [Language families](/learn/intro-software-dev/language-families/)

**Object-oriented** — A style that bundles data and the behaviour that acts on it into objects. See [Language families](/learn/intro-software-dev/language-families/)

**Functional** — A style that builds programs from functions and favours immutable data and avoiding side effects. See [Language families](/learn/intro-software-dev/language-families/)

**Declarative** — A style where you describe *what* you want rather than the steps to compute it, as in a database query. See [Language families](/learn/intro-software-dev/language-families/)

**Multi-paradigm** — A language that supports more than one paradigm, letting you mix styles as it suits the problem. See [Language families](/learn/intro-software-dev/language-families/)

## How code runs

**Compiled language** — A language whose programs are translated to machine code ahead of time, then run directly by the processor. See [Compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/)

**Interpreted language** — A language whose programs are read and executed on the fly by an interpreter, with no separate build step. See [Compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/)

**Bytecode** — A compact, portable intermediate form between source code and machine code, run by a virtual machine. See [Compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/)

**Virtual machine** — A software layer that executes bytecode, letting the same program run on different hardware. See [Compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/)

**JIT (just-in-time compilation)** — Compiling code to machine code while the program runs, combining the portability of bytecode with much of the speed of compiled code. See [Compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/)

**Ahead-of-time (AOT) compilation** — Compiling a program fully before it runs, so no compilation happens at run time. See [Compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/)

**Managed language** — A language that runs on a runtime which handles services like memory management and safety for you, rather than leaving them to the programmer. See [A tour of the languages](/learn/intro-software-dev/language-tour/)

## Types and memory

**Static typing** — Types are checked before the program runs, catching many mistakes at compile time. See [Type systems](/learn/intro-software-dev/type-systems/)

**Dynamic typing** — Types are checked while the program runs, giving flexibility at the cost of later error detection. See [Type systems](/learn/intro-software-dev/type-systems/)

**Strong vs weak typing** — How strictly a language stops you from mixing incompatible types; strong typing resists silent conversions, weak typing allows more of them. See [Type systems](/learn/intro-software-dev/type-systems/)

**Type inference** — When the language works out a value's type for you, so you do not have to write it explicitly. See [Type systems](/learn/intro-software-dev/type-systems/)

**Type safety** — The guarantee that operations only run on compatible types, preventing whole classes of bugs. See [Type systems](/learn/intro-software-dev/type-systems/)

**Gradual typing** — Mixing static and dynamic typing in one program, adding type checks where they help and leaving them out elsewhere. See [Type systems](/learn/intro-software-dev/type-systems/)

**Stack** — A region of memory used for short-lived data like function calls and local variables, managed automatically in last-in first-out order. See [Memory management](/learn/intro-software-dev/memory-management/)

**Heap** — A region of memory for data whose size or lifetime is not known in advance, allocated and freed more flexibly than the stack. See [Memory management](/learn/intro-software-dev/memory-management/)

**Pointer** — A value that holds the memory address of something else, letting code refer to data indirectly. See [Memory management](/learn/intro-software-dev/memory-management/)

**Manual memory management** — When the programmer explicitly allocates and frees memory, with full control but more room for error. See [Memory management](/learn/intro-software-dev/memory-management/)

**Garbage collection** — Automatic reclaiming of memory the program no longer uses, so the programmer does not free it by hand. See [Memory management](/learn/intro-software-dev/memory-management/)

**Memory leak** — Memory that is allocated but never released, slowly consuming resources until the program slows or crashes. See [Memory management](/learn/intro-software-dev/memory-management/)

**Ownership** — A model where each piece of data has a clear owner responsible for its lifetime, used to manage memory safely without a garbage collector. See [Memory management](/learn/intro-software-dev/memory-management/)

**Borrow checker** — A compiler feature (notably in Rust) that enforces ownership and borrowing rules to prevent memory errors before the program runs. See [Memory management](/learn/intro-software-dev/memory-management/)

## Concurrency

**Concurrency** — Structuring a program so several tasks make progress in overlapping time periods, whether or not they truly run at the same instant. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Parallelism** — Actually running multiple tasks at the same time, typically on multiple processor cores. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Thread** — An independent path of execution within a program; multiple threads can run concurrently and share memory. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Async/await** — Language syntax for writing concurrent code that waits for slow operations without blocking, in a style that reads like ordinary sequential code. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Goroutine** — A lightweight, cheap unit of concurrent execution in Go, many of which can run at once. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Channel** — A typed conduit for passing values safely between concurrent tasks, coordinating them by communication rather than shared memory. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Actor model** — A concurrency approach where independent actors hold private state and interact only by sending messages. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Race condition** — A bug where the result depends on the unpredictable timing of concurrent tasks accessing shared data. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

**Deadlock** — A standstill where tasks each wait for a resource another holds, so none can ever proceed. See [Concurrency models](/learn/intro-software-dev/concurrency-models/)

## Security

**Memory safety** — A property that prevents bugs like accessing memory you should not, which are a major source of security vulnerabilities. See [Language security](/learn/intro-software-dev/language-security/)

**Undefined behaviour** — Code whose result the language does not define, which may work by luck but can fail unpredictably or open security holes. See [Language security](/learn/intro-software-dev/language-security/)

**Buffer overflow** — Writing past the end of a block of memory, corrupting nearby data and a classic route to security exploits. See [Language security](/learn/intro-software-dev/language-security/)

**Dependency** — External code your program relies on, pulled in so you do not have to write it yourself. See [Language security](/learn/intro-software-dev/language-security/)

**Supply chain** — The full set of tools, libraries, and dependencies that go into building your software, each of which can introduce risk. See [Language security](/learn/intro-software-dev/language-security/)

**CVE (Common Vulnerabilities and Exposures)** — A publicly catalogued, uniquely identified security flaw in a piece of software. See [Language security](/learn/intro-software-dev/language-security/)

**Sandboxing** — Running code in a restricted environment so it cannot harm the rest of the system even if it misbehaves. See [Language security](/learn/intro-software-dev/language-security/)

## Principles of good code

**Readability** — How easily a human can understand code; the primary thing clean code optimises for, since code is read far more than written. See [Writing readable code](/learn/intro-software-dev/clean-code/)

**Naming** — Choosing clear, intention-revealing names for variables, functions, and types so code explains itself. See [Writing readable code](/learn/intro-software-dev/clean-code/)

**Code smell** — A surface sign that something may be wrong with the design, hinting at a deeper problem worth a closer look. See [Writing readable code](/learn/intro-software-dev/clean-code/)

**DRY (Don't Repeat Yourself)** — Avoid duplicating knowledge; keep each piece of logic in one place so changes happen once. See [DRY, KISS, and YAGNI](/learn/intro-software-dev/dry-kiss-yagni/)

**KISS (Keep It Simple, Stupid)** — Prefer the simplest solution that works; complexity should be earned, not added by default. See [DRY, KISS, and YAGNI](/learn/intro-software-dev/dry-kiss-yagni/)

**YAGNI (You Aren't Gonna Need It)** — Don't build features or flexibility on speculation; add them when a real need appears. See [DRY, KISS, and YAGNI](/learn/intro-software-dev/dry-kiss-yagni/)

**Separation of concerns** — Splitting a system so each part handles one distinct responsibility, keeping concerns from tangling together. See [DRY, KISS, and YAGNI](/learn/intro-software-dev/dry-kiss-yagni/)

**SOLID** — A set of five object-oriented design principles that together encourage flexible, maintainable code. See [SOLID](/learn/intro-software-dev/solid/)

**Single responsibility** — The idea that a unit of code should have one reason to change, that is, one clear job. See [SOLID](/learn/intro-software-dev/solid/)

**Open/closed** — Code should be open to extension but closed to modification, so you add behaviour without rewriting what works. See [SOLID](/learn/intro-software-dev/solid/)

**Liskov substitution** — A subtype should be usable anywhere its base type is expected, without breaking the program. See [SOLID](/learn/intro-software-dev/solid/)

**Interface segregation** — Prefer several small, focused interfaces over one large one, so users depend only on what they need. See [SOLID](/learn/intro-software-dev/solid/)

**Dependency inversion** — Depend on abstractions rather than concrete details, so high-level code is not tied to low-level specifics. See [SOLID](/learn/intro-software-dev/solid/)

**Dependency injection** — Supplying a component's dependencies from outside rather than having it create them, making code easier to test and swap. See [SOLID](/learn/intro-software-dev/solid/)

**Abstraction** — Hiding details behind a simpler interface so you can use something without knowing how it works inside. See [Abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/)

**Encapsulation** — Bundling data with the code that uses it and hiding the internals, so outside code interacts only through a defined surface. See [Abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/)

**Interface** — The agreed surface through which one piece of code talks to another, independent of the implementation behind it. See [Abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/)

**Coupling** — How dependent two parts of a system are on each other; loose coupling makes change safer and easier. See [Abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/)

**Cohesion** — How well the parts of a single unit belong together; high cohesion means a component does one thing well. See [Abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/)

**Leaky abstraction** — An abstraction that forces you to understand the details it was meant to hide. See [Abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/)

## Robustness and errors

**Error handling** — How a program detects, reports, and recovers from things going wrong. See [Robustness and errors](/learn/intro-software-dev/robustness-and-errors/)

**Exception** — A signal that an error occurred, which interrupts normal flow so it can be caught and handled elsewhere. See [Robustness and errors](/learn/intro-software-dev/robustness-and-errors/)

**Edge case** — An unusual or extreme input or situation at the boundary of what code expects, where bugs often hide. See [Robustness and errors](/learn/intro-software-dev/robustness-and-errors/)

**Defensive programming** — Writing code that anticipates bad input and misuse, checking assumptions rather than trusting them. See [Robustness and errors](/learn/intro-software-dev/robustness-and-errors/)

**Fail fast** — Detecting problems and stopping immediately, rather than continuing in a broken state that is harder to diagnose. See [Robustness and errors](/learn/intro-software-dev/robustness-and-errors/)

**Graceful degradation** — Continuing to provide reduced but useful service when part of a system fails, instead of collapsing entirely. See [Robustness and errors](/learn/intro-software-dev/robustness-and-errors/)

**Idempotency** — A property where doing an operation more than once has the same effect as doing it once, which makes retries safe. See [Robustness and errors](/learn/intro-software-dev/robustness-and-errors/)

## Data structures & algorithms

**Data structure** — A way of organizing data in memory so the operations you need are cheap; the right choice makes code simpler and faster. See [Data structures & algorithms, briefly](/learn/intro-software-dev/data-structures-algorithms/)

**Algorithm** — A step-by-step recipe for solving a problem, such as searching or sorting a collection. See [Data structures & algorithms, briefly](/learn/intro-software-dev/data-structures-algorithms/)

**Big-O notation** — A shorthand for how an algorithm's work grows with input size — O(1), O(n), O(n log n), O(n²) — used to compare approaches at scale. See [Data structures & algorithms, briefly](/learn/intro-software-dev/data-structures-algorithms/)

**Hash map (hash table)** — A structure that stores key→value pairs for near-instant (O(1)) lookup by key. See [Data structures & algorithms, briefly](/learn/intro-software-dev/data-structures-algorithms/)

**Ring buffer** — A fixed-size circular buffer that overwrites its oldest data, ideal for streaming a continuous flow of samples. See [Data structures & algorithms, briefly](/learn/intro-software-dev/data-structures-algorithms/)

## Design patterns

**Design pattern** — A reusable, named solution to a problem that recurs in software design. See [What are design patterns?](/learn/intro-software-dev/what-are-patterns/)

**Anti-pattern** — A common response to a problem that looks helpful but causes more harm than good. See [What are design patterns?](/learn/intro-software-dev/what-are-patterns/)

**Gang of Four** — The four authors of the influential 1994 book that catalogued the classic object-oriented design patterns. See [What are design patterns?](/learn/intro-software-dev/what-are-patterns/)

**Boilerplate** — Repetitive setup code you must write that carries little unique meaning of its own. See [What are design patterns?](/learn/intro-software-dev/what-are-patterns/)

**Factory** — A creational pattern that creates objects through a dedicated method, hiding which concrete type is built. See [Creational patterns](/learn/intro-software-dev/creational-patterns/)

**Builder** — A creational pattern that constructs a complex object step by step, separating how it is built from what it becomes. See [Creational patterns](/learn/intro-software-dev/creational-patterns/)

**Singleton** — A creational pattern that ensures a class has only one instance and provides a single point of access to it. See [Creational patterns](/learn/intro-software-dev/creational-patterns/)

**Adapter** — A structural pattern that wraps one interface so it looks like another, letting incompatible parts work together. See [Structural patterns](/learn/intro-software-dev/structural-patterns/)

**Facade** — A structural pattern that puts a simple front over a complex subsystem, giving callers an easier way in. See [Structural patterns](/learn/intro-software-dev/structural-patterns/)

**Decorator** — A structural pattern that adds behaviour to an object by wrapping it, without changing the original. See [Structural patterns](/learn/intro-software-dev/structural-patterns/)

**Observer** — A behavioural pattern where objects subscribe to another object and are notified automatically when it changes. See [Behavioural patterns](/learn/intro-software-dev/behavioral-patterns/)

**Strategy** — A behavioural pattern that lets you swap interchangeable algorithms behind a common interface at run time. See [Behavioural patterns](/learn/intro-software-dev/behavioral-patterns/)

**State pattern** — A behavioural pattern where an object changes its behaviour as its internal state changes, as if changing class. See [Behavioural patterns](/learn/intro-software-dev/behavioral-patterns/)

**Publish/subscribe** — A messaging pattern where publishers send events without knowing who, if anyone, is listening, and subscribers receive the events they care about. See [Behavioural patterns](/learn/intro-software-dev/behavioral-patterns/)

## Concurrency and architecture patterns

**Pipeline** — A design where data flows through a series of stages, each doing one step of the processing. See [Concurrency and pipelines](/learn/intro-software-dev/concurrency-and-pipelines/)

**Producer/consumer** — A pattern where producers create work items and consumers process them, usually buffered by a queue between them. See [Concurrency and pipelines](/learn/intro-software-dev/concurrency-and-pipelines/)

**Backpressure** — A mechanism that lets a slow consumer signal a fast producer to slow down, preventing work from piling up uncontrollably. See [Concurrency and pipelines](/learn/intro-software-dev/concurrency-and-pipelines/)

**Message queue** — A buffer that holds messages between parts of a system so senders and receivers do not have to act at the same moment. See [Concurrency and pipelines](/learn/intro-software-dev/concurrency-and-pipelines/)

**Layered architecture** — Organising a system into stacked layers (such as presentation, logic, data) where each layer talks mainly to the one below. See [Architectural patterns](/learn/intro-software-dev/architectural-patterns/)

**Event-driven** — An architecture where components react to events as they happen rather than being called in a fixed sequence. See [Architectural patterns](/learn/intro-software-dev/architectural-patterns/)

**Plugin architecture** — A design where a core program can be extended by add-on modules without changing the core itself. See [Architectural patterns](/learn/intro-software-dev/architectural-patterns/)

**MVC (Model-View-Controller)** — A pattern that separates data (model), presentation (view), and input handling (controller) so each can change independently. See [Architectural patterns](/learn/intro-software-dev/architectural-patterns/)

**Monolith** — An application built and deployed as a single unit, with all its parts in one codebase. See [Architectural patterns](/learn/intro-software-dev/architectural-patterns/)

**Microservices** — An architecture that splits an application into small, independently deployable services that communicate over a network. See [Architectural patterns](/learn/intro-software-dev/architectural-patterns/)

## Process, methodology and delivery

**SDLC (software development life cycle)** — The overall sequence of stages a project moves through, from idea and requirements to delivery and maintenance. See [The SDLC and methodologies](/learn/intro-software-dev/sdlc-and-methodologies/)

**Waterfall** — A sequential approach where each phase is completed before the next begins, with little going back. See [The SDLC and methodologies](/learn/intro-software-dev/sdlc-and-methodologies/)

**Agile** — A flexible approach that delivers software in small increments and adapts to change instead of following a fixed plan. See [The SDLC and methodologies](/learn/intro-software-dev/sdlc-and-methodologies/)

**Scrum** — A popular Agile framework that organises work into fixed-length sprints with defined roles and ceremonies. See [The SDLC and methodologies](/learn/intro-software-dev/sdlc-and-methodologies/)

**Kanban** — An Agile method that visualises work on a board and limits how much is in progress at once to keep flow steady. See [The SDLC and methodologies](/learn/intro-software-dev/sdlc-and-methodologies/)

**Sprint** — A short, fixed time box (often one to four weeks) in which a Scrum team completes a planned chunk of work. See [The SDLC and methodologies](/learn/intro-software-dev/sdlc-and-methodologies/)

**MVP (minimum viable product)** — The smallest version of a product that delivers real value and can be learned from. See [The SDLC and methodologies](/learn/intro-software-dev/sdlc-and-methodologies/)

**Version control** — A system that records changes to code over time, letting you review history, recover old states, and collaborate safely. See [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)

**Repository** — The store of a project's files together with their full change history under version control. See [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)

**Commit** — A saved snapshot of changes in version control, with a message describing what changed and why. See [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)

**Branch** — A separate line of development where you can work without affecting the main version until you are ready. See [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)

**Merge** — Combining the changes from one branch into another. See [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)

**Pull request** — A proposal to merge your changes, opened so others can review and discuss them first. See [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)

**Code review** — Having another developer read proposed changes to catch problems and share knowledge before they merge. See [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)

## Testing

**Unit test** — An automated test that checks one small piece of code in isolation. See [Testing](/learn/intro-software-dev/testing/)

**Integration test** — A test that checks that several parts work correctly together. See [Testing](/learn/intro-software-dev/testing/)

**End-to-end test** — A test that exercises the whole system the way a real user would, from start to finish. See [Testing](/learn/intro-software-dev/testing/)

**Regression test** — A test that guards against a previously fixed bug coming back. See [Testing](/learn/intro-software-dev/testing/)

**TDD (test-driven development)** — A practice of writing a failing test first, then writing just enough code to make it pass. See [Testing](/learn/intro-software-dev/testing/)

**Mock** — A stand-in for a real dependency in a test, used to isolate the code under test and control its surroundings. See [Testing](/learn/intro-software-dev/testing/)

**Code coverage** — A measure of how much of your code your tests actually exercise. See [Testing](/learn/intro-software-dev/testing/)

**Golden file** — A saved reference output that a test compares against to catch unintended changes. See [Testing](/learn/intro-software-dev/testing/)

## Build and delivery

**Build system** — Tooling that turns source code into a runnable program, handling compilation and assembly steps. See [Builds and CI/CD](/learn/intro-software-dev/build-and-ci-cd/)

**Dependency management** — Declaring, fetching, and tracking the external libraries a project relies on, including their versions. See [Builds and CI/CD](/learn/intro-software-dev/build-and-ci-cd/)

**Continuous integration (CI)** — Automatically building and testing every change as it is merged, so problems surface early. See [Builds and CI/CD](/learn/intro-software-dev/build-and-ci-cd/)

**Continuous delivery (CD)** — Automating the steps to release software so a tested change can be shipped quickly and reliably. See [Builds and CI/CD](/learn/intro-software-dev/build-and-ci-cd/)

**Artifact** — A built output of the process, such as a binary or package, that can be stored and deployed. See [Builds and CI/CD](/learn/intro-software-dev/build-and-ci-cd/)

**Cross-compilation** — Building a program on one type of machine to run on a different type, such as compiling on a laptop for a small device. See [Builds and CI/CD](/learn/intro-software-dev/build-and-ci-cd/)

## Documentation and maintenance

**Documentation** — Written explanation of how software works and how to use it, for users and future maintainers. See [Documentation and maintenance](/learn/intro-software-dev/docs-and-maintenance/)

**Refactoring** — Improving the internal structure of code without changing what it does, to keep it clean as it grows. See [Documentation and maintenance](/learn/intro-software-dev/docs-and-maintenance/)

**Technical debt** — The future cost of taking shortcuts now; like financial debt, it accrues interest until you pay it down. See [Documentation and maintenance](/learn/intro-software-dev/docs-and-maintenance/)

**Deprecation** — Marking a feature as outdated and discouraged, signalling it will be removed so users can move off it. See [Documentation and maintenance](/learn/intro-software-dev/docs-and-maintenance/)

**Bus factor** — The number of people who would have to be lost before a project stalls; a bus factor of one is a warning sign. See [Documentation and maintenance](/learn/intro-software-dev/docs-and-maintenance/)

## Data, storage & APIs

**Persistence** — Keeping data so it survives a program exiting or the power cycling, by writing it to a file or database. See [Storing data — files & databases](/learn/intro-software-dev/databases-and-persistence/)

**Database** — An organized store designed for querying and updating data reliably, from a single-file embedded engine to a networked server. See [Storing data — files & databases](/learn/intro-software-dev/databases-and-persistence/)

**SQL vs NoSQL** — SQL databases store structured rows in related tables with strong guarantees; NoSQL stores (key-value, document, time-series) trade some guarantees for flexibility. See [Storing data — files & databases](/learn/intro-software-dev/databases-and-persistence/)

**SQLite** — A self-contained database that lives in a single file with no server, the common choice for desktop and embedded apps. See [Storing data — files & databases](/learn/intro-software-dev/databases-and-persistence/)

**API (application programming interface)** — A defined contract that lets one piece of software use another without knowing its internals. See [APIs & talking to other software](/learn/intro-software-dev/apis-and-integration/)

**REST / HTTP / JSON** — The common style and formats of web APIs: resources addressed by URL, HTTP methods as verbs, and JSON as the data format. See [APIs & talking to other software](/learn/intro-software-dev/apis-and-integration/)

## Choosing a language

**Lock-in** — Becoming so tied to a particular technology that moving away from it is costly or impractical. See [Why language choice matters](/learn/intro-software-dev/why-language-choice-matters/)

**Switching cost** — The effort, time, and risk involved in moving from one technology to another. See [Why language choice matters](/learn/intro-software-dev/why-language-choice-matters/)

**Ecosystem** — The libraries, tools, and community around a language that make it more or less productive in practice. See [Why language choice matters](/learn/intro-software-dev/why-language-choice-matters/)

**Requirements** — What a system must actually do and the conditions it must meet, gathered before deciding how to build it. See [Starting from requirements](/learn/intro-software-dev/from-requirements/)

**Constraint** — A fixed limit the solution must respect, such as a deadline, budget, platform, or performance target. See [Starting from requirements](/learn/intro-software-dev/from-requirements/)

**Real-time** — A requirement that a system respond within strict, guaranteed time limits, where being late is itself a failure. See [Starting from requirements](/learn/intro-software-dev/from-requirements/)

**Performance** — How fast and efficiently a program runs with the resources it has. See [Performance vs productivity](/learn/intro-software-dev/performance-vs-productivity/)

**Productivity** — How quickly and easily developers can build and change software, often traded off against raw performance. See [Performance vs productivity](/learn/intro-software-dev/performance-vs-productivity/)

**Premature optimization** — Tuning for speed before you know it matters, adding complexity that often is not needed. See [Performance vs productivity](/learn/intro-software-dev/performance-vs-productivity/)

**Native library** — Precompiled code, often written in a lower-level language, that a program calls for speed or to reach system features. See [Performance vs productivity](/learn/intro-software-dev/performance-vs-productivity/)

**Domain** — The particular problem area or field a piece of software serves, which shapes what tools fit best. See [Choosing a language for the domain](/learn/intro-software-dev/language-for-the-domain/)

**Systems programming** — Writing low-level software like operating systems, drivers, and runtimes, where control and performance matter most. See [Choosing a language for the domain](/learn/intro-software-dev/language-for-the-domain/)

**Embedded** — Software that runs on small, resource-limited devices dedicated to a specific job rather than general computing. See [Choosing a language for the domain](/learn/intro-software-dev/language-for-the-domain/)

**Scripting** — Writing short, often interpreted programs to automate tasks or glue other tools together. See [Choosing a language for the domain](/learn/intro-software-dev/language-for-the-domain/)

**Decision framework** — A structured way to weigh options against your needs so a choice is reasoned rather than arbitrary. See [A decision framework](/learn/intro-software-dev/decision-framework/)

**Prototype** — A quick, throwaway or rough build made to test an idea or reduce uncertainty before committing to it. See [A decision framework](/learn/intro-software-dev/decision-framework/)

**Trade-off** — A choice where gaining one quality means giving up another, such as speed versus simplicity. See [A decision framework](/learn/intro-software-dev/decision-framework/)

## Going solo and shipping

**Solo developer** — Someone who builds and maintains software largely on their own, wearing every role at once. See [The solo developer mindset](/learn/intro-software-dev/solo-developer-mindset/)

**Tech stack** — The combined set of languages, frameworks, and tools chosen to build a project. See [Choosing your stack](/learn/intro-software-dev/choosing-your-stack/)

**Framework** — A reusable foundation that provides structure and calls your code, shaping how an application is built. See [Choosing your stack](/learn/intro-software-dev/choosing-your-stack/)

**Library** — A collection of reusable code you call from your own program to avoid reinventing common functionality. See [Choosing your stack](/learn/intro-software-dev/choosing-your-stack/)

**IDE (integrated development environment)** — An editor bundled with tools like building, debugging, and code navigation in one place. See [The development environment workflow](/learn/intro-software-dev/dev-environment-workflow/)

**Linter** — A tool that scans code for likely mistakes and style problems without running it. See [The development environment workflow](/learn/intro-software-dev/dev-environment-workflow/)

**Formatter** — A tool that automatically rewrites code into a consistent style, ending arguments over layout. See [The development environment workflow](/learn/intro-software-dev/dev-environment-workflow/)

**Dotfiles** — Personal configuration files for your tools and shell, often kept in version control to reproduce your setup. See [The development environment workflow](/learn/intro-software-dev/dev-environment-workflow/)

**Devcontainer** — A defined, containerised development environment so everyone (including future you) works with the same setup. See [The development environment workflow](/learn/intro-software-dev/dev-environment-workflow/)

**Build-test loop** — The fast cycle of changing code, building it, and running tests that drives day-to-day development. See [The development environment workflow](/learn/intro-software-dev/dev-environment-workflow/)

**Scope creep** — The gradual, unplanned growth of what a project is meant to do, which threatens deadlines and focus. See [Scoping a project](/learn/intro-software-dev/scoping-a-project/)

**Milestone** — A meaningful checkpoint in a project marking that a defined chunk of work is complete. See [Scoping a project](/learn/intro-software-dev/scoping-a-project/)

**Specification** — A clear written statement of what is to be built and how it should behave. See [Scoping a project](/learn/intro-software-dev/scoping-a-project/)

**Pre-commit hook** — An automated check that runs before a commit is recorded, blocking obvious problems early. See [Quality when you are solo](/learn/intro-software-dev/quality-when-solo/)

**Guardrail** — An automated check or constraint that keeps you from making certain mistakes, even when no one is reviewing. See [Quality when you are solo](/learn/intro-software-dev/quality-when-solo/)

## Packaging and distribution

**Binary** — A compiled, runnable file produced from source code, ready to execute on a target machine. See [Packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/)

**Semantic versioning** — A version-numbering scheme (major.minor.patch) that signals the nature of each change to users. See [Packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/)

**Release** — A specific, packaged version of software made available for people to use. See [Packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/)

**Package manager** — A tool that installs, updates, and manages software and its dependencies for you. See [Packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/)

**Code signing** — Cryptographically marking software so users can verify who produced it and that it has not been tampered with. See [Packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/)

**Checksum** — A short value computed from a file that lets you verify the file downloaded intact and unaltered. See [Packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/)

**Release notes** — A summary of what changed in a release, telling users what is new, fixed, or removed. See [Shipping and maintaining](/learn/intro-software-dev/shipping-and-maintaining/)

**Maintenance** — The ongoing work of keeping released software working: fixing bugs, updating dependencies, and adapting to change. See [Shipping and maintaining](/learn/intro-software-dev/shipping-and-maintaining/)
