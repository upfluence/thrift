namespace * types.annotation.text

// Format defines the canonical text form of a struct, used to encode it as a
// string (JSON, SQL, CLI flags...). The text codecs of the thrift libraries
// encode and decode the struct following it.
//
// The pattern follows the python format string syntax, without format specs:
// "{field_name}" placeholders and literal separators, "{{" and "}}" being
// escaped braces, e.g. "{type}:{entity_id}". The pattern must reference every
// field of the struct exactly once, and two placeholders must be separated by
// a literal. A field is decoded up to the first occurrence of the literal
// following it, so encoding a string value containing that literal fails.
// Fields must not be optional, and must be typed as an enum, an integer or a
// string.
//
// Placement:
//   - On a struct only.
struct Format {
  1: required string pattern;
}

// Name overrides the text form of an enum value. By default, the text form
// of a value is derived from its name, see EnumFormat.
//
// Placement:
//   - On an enum value only.
struct Name {
  1: required string value;
}

// Case is the convention used to derive the text form of an enum value from
// its name. The name is split into words on underscores, on lower to upper
// case changes and at the end of acronyms: PaypalAPI is "Paypal API",
// HTTPServer is "HTTP Server".
enum Case {
  // PaypalAPI is "paypal_api".
  SnakeCase          = 0,
  // PaypalAPI is "paypal-api".
  KebabCase          = 1,
  // PaypalAPI is "PAYPAL_API".
  ScreamingSnakeCase = 2,
  // PaypalAPI is "paypalAPI", the Go initialisms are upper cased.
  CamelCase          = 3,
  // PaypalAPI is "PaypalAPI", the Go initialisms are upper cased.
  PascalCase         = 4,
  // PaypalAPI is "paypalapi".
  LowerCase          = 5,
  // PaypalAPI is "PaypalAPI", the name is kept as is.
  Identity           = 6,
}

// EnumFormat sets how the text form of an enum value is derived from its
// name when the value has no Name annotation. The first prefix of
// trim_prefixes, and the first suffix of trim_suffixes, matching the thrift
// name are trimmed, in list order, then the case is applied. Without
// EnumFormat, the text form of a value is the snake case of its name.
//
// Only the text form of a value is decoded, unless accept_thrift_names is
// set: the thrift names of the value ("Company" and "Type_Company" for the
// Company value of the Type enum) are decoded as well, e.g. to read data
// written before the enum was annotated.
//
// Placement:
//   - On an enum only.
struct EnumFormat {
  1: optional Case         format = Case.SnakeCase;
  2: optional list<string> trim_prefixes;
  3: optional list<string> trim_suffixes;
  4: optional bool         accept_thrift_names;
}
