namespace * types.annotation.rpc

// Idempotent marks an RPC function (or all functions of a service) as
// idempotent: calling the function multiple times with the same arguments
// produces the same result as calling it once. Idempotency is a contract
// between the service and its callers; clients may safely retry an idempotent
// call after a transient failure without risking unintended state changes.
//
// Placement:
//   - On a function: only that specific function is considered idempotent.
//   - On a service: every function in the service is considered idempotent.
//
// Note: an idempotent operation may still produce side effects (e.g. writing
// a record that is deduplicated by a unique key). If an operation has NO side
// effects at all, prefer ReadOnly instead, which carries the stronger
// guarantee that concurrent calls are safe.
struct Idempotent {}

// ReadOnly marks an RPC function (or all functions of a service) as
// read-only: the function neither mutates state nor produces any observable
// side effect. Because a read-only call leaves the system unchanged, it is
// always idempotent and is additionally safe to execute concurrently — callers
// may issue multiple in-flight requests in parallel without risk of
// inconsistency.
//
// Placement:
//   - On a function: only that specific function is considered read-only.
//   - On a service: every function in the service is considered read-only.
//
// Relationship to Idempotent:
//   ReadOnly ⊂ Idempotent. Every read-only function is implicitly idempotent,
//   but the converse is not true — an idempotent function may still write
//   state (e.g. an upsert). Use ReadOnly when no side effects exist; use
//   Idempotent when side effects are present but repeated calls are safe.
@Idempotent{}
struct ReadOnly {}
