namespace go test.plugin

struct Foo {
  1: required string name
}

struct Bar {
  1: required i32 id
}

exception Baz {
  1: required string message
}

service Qux {
  void ping()
  Foo get(1: required i32 id)
}
