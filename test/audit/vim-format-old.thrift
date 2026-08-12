namespace * audit

struct Delta {
  3: required string id;
  4: optional string old_name;
  5: optional string value;
  6: optional string removed;
}
