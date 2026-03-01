# Memory Direction — cx

## Always Save
- **Performance characteristics**: Benchmark results, measured latencies, throughput characteristics
- **Data model and migration history**: Schema changes, migration gotchas, data integrity constraints
- **Security decisions and tradeoffs**: Auth decisions, encryption choices, vulnerability mitigations
- **API contracts**: Any change to request/response format
- **Database gotchas**: Query performance issues, locking behavior

## Never Save
- **Duplicated issue content**: Anything already captured in Linear — don't duplicate issue descriptions
- **Standard library behavior**: Official documentation already covers this
- **Code-visible implementation**: Details visible by reading the source file itself
- **Status updates**: Use session summaries for progress tracking, not observations
- **Framework defaults**: Behavior that's documented in the framework's official docs

## Threshold Test
When unsure, ask:
- Would a new team member need to know this to avoid hitting the same wall?
- Is this a constraint that can't be inferred from the code or docs?
- Would I be frustrated to rediscover this in 3 months?

## Type Guidance
- **bugfix**: API errors, database query issues, authentication failures
- **discovery**: Third-party API constraints, database behavior under load
- **pattern**: Middleware patterns, error handling conventions, testing approaches
- **context**: Deployment environment specifics, integration partner requirements
