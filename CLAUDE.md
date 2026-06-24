# Project overview
This project aggregates academic publications from RSS feeds into a unified local database that can be browsed, filtered, and automatically ranked by relevance to the user's personal research interests.

**The central feature is relevance scoring**: every ingested publication is analyzed against the user's interest profile and assigned a score (0–1). Two interchangeable scorers are supported — a pure-Go keyword scorer (offline, no deps) and an LLM scorer (any OpenAI-compatible endpoint, semantic understanding). This is what makes paper-inator useful rather than just being another RSS reader.

It has three main parts:
- `/src/serviceWorker`: pulls/parses feeds, maps fields, deduplicates publications, scores relevance, stores data, generates email summaries
- `/src/frontend`: pure HTML/CSS/JS web UI for managing feeds, mappings, publications, relevance settings, and summaries
- `/src/api`: REST API for external clients and for frontend/backend configuration

Primary goal: easy self-hosted deployment for non-expert scientists. The preferred deployment model is a single executable behind nginx with SQLite as the only required database.

## Technical constraints
- Prefer a performant backend language with minimal dependencies.
- Prefer solutions that can compile to a single executable.
- Use SQLite for persistence.
- Frontend must remain pure HTML/CSS/JS.
- API must remain RESTful and stable.
- Do not introduce large frameworks or infrastructure-heavy dependencies without explicit approval.

## Architecture
- Service worker owns ingestion, parsing, normalization, deduplication, and summary generation.
- API owns external access to publications, feeds, mappings, summaries, and persisted settings.
- Frontend owns human-facing CRUD and filtering UI.
- Shared types and validation logic should be reused where practical instead of duplicated.

## Default workflow
- For any non-trivial task, start in Plan Mode.
- Before editing, inspect the existing implementation and produce a plan covering:
  1. files to change
  2. data model impact
  3. API impact
  4. frontend impact
  5. migration needs
  6. tests to add or update
  7. risks / trade-offs
- Do not edit files until the plan is approved.

## Commit workflow
- Before starting any implementation, plan the work in commits (what logical unit each commit will cover).
- Commit as work progresses, not all at once at the end.
- After completing each commit-worthy unit, propose the commit to the user with a ready-to-use commit message. Only commit if the user approves.
- Split work into small, reviewable commits: one logical change per commit (e.g. "add data model", "add store layer", "add API handlers" are separate commits, not one big "implement feature X").
- Never commit directly to `main`. All work happens on a feature branch.

## Change rules
- Never commit directly to `main`.
- Create a branch for each feature or fix.
- Prefer small, reviewable patches.
- Preserve backward compatibility unless a breaking change is explicitly requested.
- Do not rename or reorganize major folders without justification.

## File placement
- Feed ingestion, parsing, field mapping, deduplication, and email summary generation belong in `/src/serviceWorker`.
- UI views and browser-side interaction logic belong in `/src/frontend`.
- REST handlers and external integration points belong in `/src/api`.
- Shared database schema definitions, models, and common helpers should live in a clearly shared location and not be reimplemented in each subproject.

## Quality bar
A task is not done unless:
- the code builds successfully
- schema changes include a migration strategy
- API changes are reflected consistently in frontend/backend usage
- feed parsing and deduplication behavior are tested
- email summary logic is tested
- user-facing settings remain persisted correctly in SQLite
- the code is modular, following the KISS and single-responsibility principles. 
- the code is easily understandable and maintainable for human programmers

## Product-specific rules
- Publications may come from heterogeneous RSS formats; field mapping must stay configurable per feed.
- Deduplication should be deterministic and explainable, based primarily on title and authors unless otherwise specified.
- Relevance scoring is the core value-add: every publication must eventually receive a score. Scoring failures must be silent and retriable — never block ingestion. Unscored publications are shown without a badge and re-queued automatically.
- The keyword scorer and LLM scorer must be interchangeable via a single settings toggle. The scorer interface must remain stable so new scorer types can be added without touching the enrichment worker.
- Email summaries must remain user-configurable by feed selection and item count.
- The system should remain understandable and operable for non-programming users.
