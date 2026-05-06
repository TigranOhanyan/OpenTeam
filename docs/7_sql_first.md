# SQL-First Persistence

## Summary

For OpenTeam, the SQLite file is not just an internal storage detail. It is the portable runtime artifact of the system. Every request loads a previously persisted SQLite file, appends the user's new message, runs the stateless engine, and persists a new version of that file.

Because of that, schema evolution is not an occasional migration concern. It is part of the normal execution path for every ongoing conversation.

## Why Code-First Is Risky Here

In a typical ORM-driven system, generating SQL schemas from Go code is convenient because the database is mostly an implementation detail. But in OpenTeam, the database file itself is part of the product contract.

If the schema is derived from code:

- A change in Go structs can accidentally invalidate persisted SQLite artifacts.
- Old conversations may stop working, because the engine depends on loading their previous SQLite state on every turn.
- Backward compatibility becomes coupled to application refactors.
- The artifact format becomes implicit rather than explicit.

This is especially dangerous in a stateless architecture, because "old files" are not rare historical records. They are the active state of current user conversations.

## Why SQL-First Fits Better

If the framework matures into a stable set of roughly 5-10 core access patterns, then the main advantage of an ORM graph API becomes less important. Writing and maintaining a limited set of explicit SQL queries is a reasonable cost in exchange for stronger control over artifact compatibility.

A SQL-first approach better matches the architecture:

- The SQLite schema becomes an explicit, versioned format.
- Compatibility is handled through deliberate migrations.
- The persisted artifact can be treated as a stable wire format for the engine.
- The system becomes easier to reason about operationally and historically.

## Recommended Direction

The persistence layer should be SQL-first:

- Define the schema with explicit SQL migrations.
- Treat the SQLite file as a versioned artifact format.
- Store a schema version in the database.
- On each request, load the prior artifact, upgrade it if needed, execute the engine, and persist a new artifact version.
- Avoid ORM auto-migration as the source of truth.

## Design Principle

Application code should adapt to the artifact format, not the other way around.

In OpenTeam, the SQLite file is effectively part of the engine API. Because of that, the schema should be owned by migrations first and application code second.
