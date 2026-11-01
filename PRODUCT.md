# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

React 19 + TypeScript + Vite 7, per `docs/design.md`. The frontend is a same-origin SPA: Vite proxies
`/api` to the Go backend in development, nginx repeats it in the container.

## Users

- **Primary: the evaluator.** A ShopBack engineer or hiring manager opening the repository, running it
  once, and judging it in a short review. They arrive cold, on a laptop, and read both the running app
  and the code.
- **In-product: one person with a list.** The demo user the reviewer becomes: registers, adds a few
  everyday tasks, ticks them off, filters. Short sessions, phone or laptop.

## Product Purpose

A small full-stack to-do application built as a take-home exercise. It exists to demonstrate a
complete, reviewable slice: HTTP API, session auth, ownership, database, tests, and a typed frontend.
Success means the reviewer can run one command, reach a working task list in under a minute, and find
nothing in the code that needs apologising for.

## Positioning

Not a product competing in a market: a working demonstration where every layer is real. Real Postgres,
real HttpOnly session cookies, no mocked API, no fake loading states. The claim it must earn is
"this person finishes things and leaves nothing hidden".

## Operating Context

- The reviewer clones, runs `docker compose up --build` (later) or the Makefile targets, and opens the
  app at localhost.
- They register a throwaway account, create tasks, toggle them, filter, log out, log back in.
- They then read the source: routes, services, repositories, tests, docs.
- Sample task content in screenshots and seeds is illustrative and must never be presented as real
  usage, testimonials, or metrics.

## Capabilities and Constraints

- Three screens: register, log in, task list. One role, no admin surface.
- Task CRUD with partial edits and an `all | active | completed` filter.
- No pagination, ordering controls, drag-and-drop, sharing, notifications, or offline mode.
- No refresh tokens: the 30-minute cookie expires and the user signs in again.
- Errors come from one backend envelope (`code`, `message`, optional `fields`); the UI must render
  those, not invent its own wording.
- Empty, loading, error, and expired-session states are part of the product, not edge cases.

## Brand Commitments

Neutral. The app uses its own name and palette and carries no ShopBack logo, wordmark, or brand colour
(confirmed with the author). The repository name references ShopBack because it is the submission
context, not a brand licence.

## Evidence on Hand

No real users, testimonials, press, or usage metrics exist and none may be fabricated. The only real
evidence is the working software and its tests. Task content used in the demo is sample data.

## Product Principles

1. **Show the whole stack working.** Every claim in the README is demonstrable in the running app.
2. **The reviewer's minute is the budget.** Reach a populated task list without onboarding or setup
   beyond the documented command.
3. **States are the product.** Empty, loading, error, and expired session are designed, not defaulted.
4. **One obvious action per screen.** Add a task, or sign in; everything else is secondary.
5. **Nothing hidden.** No mocked data paths, no dead controls, no comments standing in for code.

## Accessibility & Inclusion

Keyboard-operable throughout with visible focus; WCAG AA contrast for text and controls; form errors
announced and tied to their fields; `prefers-reduced-motion` respected for any motion.
