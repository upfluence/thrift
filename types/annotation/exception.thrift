namespace * types.annotation.exception

// Safe marks an exception as safe to retry: when a client receives an
// exception decorated with this annotation, it may re-issue the original
// request without risk of unintended side effects or data corruption.
//
// Placement:
//   - On an exception struct only. Applying this annotation to any other
//     Thrift construct (struct, service, function, field, …) has no defined
//     meaning and should be treated as an error by tooling.
//
// Retry semantics:
//   A Safe exception signals that the server did not (or only partially)
//   process the request before raising the exception, so retrying is
//   equivalent to the first attempt. Typical examples include:
//     - Transient resource exhaustion (rate limit exceeded, back-pressure)
//     - Temporary unavailability of a downstream dependency
//     - Request rejected before any mutation was applied
//
//   Callers must NOT retry exceptions that are NOT annotated with Safe,
//   unless they have out-of-band knowledge that the underlying operation is
//   idempotent (see types.annotation.rpc.Idempotent / ReadOnly).
//
// Relationship to rpc annotations:
//   Safe on an exception and Idempotent/ReadOnly on a function are
//   complementary but orthogonal. An Idempotent function may still raise
//   non-Safe exceptions (e.g. a validation error that is never worth
//   retrying). Conversely, a Safe exception raised by a non-idempotent
//   function is still safe to retry because the server guarantees it did not
//   apply any mutation before throwing.
struct Safe {}

enum Kind {
  Unknown = 0,
  Transient = 1,
  Permanent = 2,
  Stateful = 3,
}

enum Blame {
  Unknown = 0,
  Client = 1,
  Server = 2,
  ThirdParty = 3,
}
