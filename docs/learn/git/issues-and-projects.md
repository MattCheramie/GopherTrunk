---
slug: issues-and-projects
title: Issues, labels & Projects
description: How to track work on GitHub with Issues — Markdown descriptions, task lists, @mentions, cross-references, labels, milestones, templates — and organise them with Projects.
keywords: github issues, issue templates, task list, github labels, milestones, github projects, project board, closing keywords, link issue pull request, assignees
level: beginner
status: full
prereq:
  - what-is-github
faq:
  - q: How do I close an issue automatically when a pull request merges?
    a: 'Put a closing keyword followed by the issue number in the pull request description or a commit message — for example "Closes #42", "Fixes #42", or "Resolves #42". When that pull request is merged into the repository''s default branch, GitHub automatically closes issue #42 and records the link. The keywords close, closes, closed, fix, fixes, fixed, resolve, resolves, and resolved all work.'
  - q: What is the difference between an issue and a GitHub Project?
    a: An issue is a single tracked item — one bug, feature request, or question — living in one repository. A GitHub Project is a planning surface that gathers many issues (and pull requests, even from different repositories) into a board or table so you can prioritise, assign custom fields like status or sprint, and see the bigger picture. Issues hold the detail; Projects organise them.
  - q: How do task lists work in a GitHub issue?
    a: 'Write a Markdown list where each item starts with "- [ ]" for unchecked or "- [x]" for checked. GitHub renders these as interactive checkboxes you can tick directly in the issue, and it shows a progress count (for example "2 of 5"). If a line references another issue like "- [ ] #17", GitHub treats it as a tracked sub-task and reflects that issue''s open/closed state.'
  - q: What are issue templates and why use them?
    a: Issue templates pre-fill the body of a new issue with a structure — for example a bug report asking for steps to reproduce, expected behaviour, and environment. They live in .github/ISSUE_TEMPLATE/ as Markdown or YAML forms. They save contributors from staring at a blank box and ensure maintainers get the information they need to act on an issue quickly.
---

# Issues, labels & Projects

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **issue** is GitHub's unit of tracked work — a bug, a feature request, or a
question — with a title, a Markdown **description**, and a discussion thread. Sharpen
issues with **task lists** (`- [ ]`), **@mentions**, and **cross-references** like
`#123`, and close them automatically with keywords such as **`Closes #42`** in a
pull request. Organise them with **labels**, **assignees**, and **milestones**, and
standardise new ones with **issue templates**. When you need the bigger picture,
**GitHub Projects** gather issues and pull requests into boards and tables with custom
fields and built-in automation.
</div>

You've seen [what GitHub adds](/learn/git/what-is-github/) on top of Git. The first
thing most teams reach for is **Issues** — the place where work is named, discussed,
and tracked before a single line of code is written. This lesson covers issues end to
end, then shows how Projects turn a pile of issues into a plan.

## Issues: the unit of tracked work

An issue is a small, focused record of something that needs attention. Three common
kinds:

- a **bug** — "Login button does nothing on Safari";
- a **feature request** — "Add dark mode to the settings page";
- a **question or discussion** — "Should we drop support for Node 16?".

Each issue has a number (unique within the repository, like `#42`), a title, an
author, a state of **open** or **closed**, and a comment thread. You create one from
the **Issues** tab with **New issue**. The number is permanent — even after the issue
closes, `#42` keeps pointing at it forever, which is what makes cross-references
reliable.

## Writing a good issue body

The description box accepts full **Markdown**, so you can use headings, code blocks,
links, and images (paste or drag a screenshot straight in). Three features turn a flat
description into something interactive:

**Task lists** break work into checkable steps. GitHub renders them as live checkboxes
and shows a progress count:

```markdown
## Steps to ship dark mode
- [x] Add a theme toggle to settings
- [ ] Persist the choice in localStorage
- [ ] Respect the OS `prefers-color-scheme` setting
- [ ] #58  ← a tracked sub-issue
```

**@mentions** pull people in: typing `@octocat` notifies that user and links their
profile. **Cross-references** connect issues and pull requests — type `#` and GitHub
offers an autocomplete of numbers and titles:

```markdown
This is blocked by #58 and duplicates the report in #12.
Probably the same root cause as octo-org/other-repo#7.
```

GitHub records these links on both ends, so opening `#58` shows that this issue
references it.

## Closing issues from commits and PRs

You don't have to close issues by hand. If a pull request (or a commit message)
contains a **closing keyword** followed by an issue number, merging that PR into the
default branch closes the issue automatically:

```text
Fix null check on empty session

Closes #42
```

The recognised keywords are `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`,
`resolve`, `resolves`, and `resolved`. One PR can close several issues
(`Closes #42, closes #43`). This is the everyday way issues and code stay in sync —
the work that fixes a bug is the thing that retires its issue.

## Labels, assignees, and milestones

Three lightweight fields give issues structure without ceremony:

| Field | What it does | Example |
|-------|--------------|---------|
| **Labels** | Coloured tags for type, priority, area | `bug`, `enhancement`, `good first issue`, `priority: high` |
| **Assignees** | Who is responsible right now | up to 10 people per issue |
| **Milestone** | A named target the issue rolls up into | `v2.0`, `Q3 cleanup` |

Labels are the workhorse — every new repo ships with a few defaults (`bug`,
`documentation`, `good first issue`) and you can add your own under **Issues → Labels**.
A **milestone** groups issues toward a goal and shows a completion bar as its issues
close, which makes it a simple release tracker. You filter the issue list on any of
these, e.g. `is:open label:bug milestone:v2.0`.

## Issue templates and forms

A blank description box invites low-quality reports. **Issue templates** pre-fill the
body with the structure you want. They live in `.github/ISSUE_TEMPLATE/` and come in
two flavours. A classic Markdown template is just a file with front matter:

```markdown
---
name: Bug report
about: Report something that isn't working
labels: ["bug"]
---

**What happened?**

**Steps to reproduce**
1.
2.

**Expected behaviour**

**Environment** (OS, browser, version):
```

The newer **issue forms** use YAML to render real input fields — dropdowns, checkboxes,
required text areas — so contributors fill a form instead of editing Markdown:

```yaml
name: Bug report
description: Report something that isn't working
labels: ["bug"]
body:
  - type: textarea
    id: what-happened
    attributes:
      label: What happened?
    validations:
      required: true
  - type: dropdown
    id: browser
    attributes:
      label: Browser
      options: [Chrome, Firefox, Safari, Edge]
```

When someone clicks **New issue**, GitHub offers a chooser listing every template.

## GitHub Projects: boards and tables

A single issue is a tree; a **Project** is the forest. Projects are flexible planning
surfaces that collect issues and pull requests — even across multiple repositories —
and let you view them as a **board** (Kanban columns like *Todo → In progress → Done*)
or a **table** (a spreadsheet-like grid). You add **custom fields** that issues don't
have on their own:

- a **Status** single-select (the columns of your board);
- a **Priority**, **Size**, or **Iteration**/sprint field;
- dates, numbers, or text for anything else you track.

Projects include **built-in automation** (workflows): for example, set an item to
*In progress* when its linked PR opens, or move it to *Done* when the issue closes —
no extra setup beyond toggling the workflow on. Because a Project item can be the same
issue you triaged earlier, **linking issues and PRs** is what ties it all together:
an issue carries its labels and discussion, the linked PR carries the code, and the
Project shows where the whole thing sits in your plan.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a closing keyword plus the issue number does it on merge." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes merging a pull request automatically close issue #42?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Adding the label <code>closed</code> to the PR</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Writing <code>Closes #42</code> in the PR description</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Assigning the PR to the issue's author</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **issue** is one tracked item — bug, feature, or question — with a number, a
  Markdown body, and a thread.
- Use **task lists** (`- [ ]`), **@mentions**, and **`#123`** cross-references to make
  issues interactive and connected.
- **Closing keywords** like `Closes #42` in a PR retire issues automatically on merge.
- **Labels**, **assignees**, and **milestones** add structure you can filter on.
- **Issue templates and forms** in `.github/ISSUE_TEMPLATE/` standardise new reports.
- **Projects** organise many issues and PRs into boards/tables with custom fields and
  automation.

Next up: turning work into code review with [pull requests](/learn/git/pull-requests/).
