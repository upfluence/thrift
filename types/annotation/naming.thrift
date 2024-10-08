namespace * types.annotation.naming

struct PreviouslyKnownAs {
  // If namespace is nil, the previous namespace is the current one
  1: optional string namespace_;
  // If name is nil, the previous struct/service name is the current one
  2: optional string name;
}
