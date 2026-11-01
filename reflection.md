# AI Usage Write-Up

## 1. How did you break down the problem before prompting?

My first prompt ([`prompt.md`](prompt.md)) was a brief, not a request for code. It fixed the stack, features and constraints, then told the AI to analyse requirements, propose an architecture, list security concerns and edge cases, and split the work into small tasks. It also said to implement one task at a time and wait for my review.

The result was ten small tasks, backend first, then frontend. The brief only needs CRUD and persistence. I added per-user accounts, so I knew the main risk was one user reaching another's tasks. I enforced that in SQL (`WHERE user_id = $1`) and tested it against a real database.

## 2. What did the AI get wrong, and how did you fix it?

- An installed Go/Gin skill recommended running `html.EscapeString` on all inbound strings. I rejected it: escaping on write corrupts stored data and double-encodes once React escapes at render time. The API stores raw text and React handles output.
- I pinned Gin to 1.11 instead of 1.12, because 1.12 pulls a MongoDB BSON driver into a Postgres-only app.
- The design skill offered two directions. I rejected the "editorial greeting" one because a to-do list is something you work through, not something that greets you.

## 3. What did you deliberately not delegate to AI, and why?

I kept the decisions I would have to defend:

- **Stack, architecture and scope.** These were in my prompt before the AI wrote a line.
- **Review gates.** I approved each task before the next started, and I approved every commit myself.
- **Security trade-offs.** I accepted a 30-minute JWT with no refresh or server-side revocation, and no rate limiting. These are written down as known limitations, not hidden.

The AI can generate options, but I am the one who has to explain them.

## 4. What would you do differently with more time?

- **Finish what is missing:** frontend tests, a lint step, rate limiting on login, and a clear "session expired" message instead of a silent redirect.
- **Use the server-side** `?status=` **filter** the backend already supports, or remove it. Right now the filter runs in the browser.