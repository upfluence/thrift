// Types exercising the text codecs of the libraries, see
// types/annotation/text.thrift.

include "types/annotation/text.thrift"

namespace * texttest

@text.EnumFormat{"accept_thrift_names": true}
enum Type {
  Company     = 0,
  Team        = 1,
  PaypalAPI   = 2,
  @text.Name{"value": "gmail"}
  GoogleOAuth = 3,
}

@text.EnumFormat{
  "format": text.Case.KebabCase,
  "trim_prefixes": ["Media"],
  "trim_suffixes": ["Short"]
}
enum MediaType {
  MediaInstagram    = 0,
  MediaYoutubeShort = 1,
  @text.Name{"value": "tt"}
  MediaTiktok       = 2,
}

enum Plain {
  FooBar = 0,
}

@text.Format{"pattern": "{type}:{entity_id}"}
struct Ownership {
  1: required Type type;
  2: required i64  entity_id;
}

@text.Format{"pattern": "{media}/{name}@{{{shard}}}"}
struct Handle {
  1: required MediaType media;
  2: required string    name;
  3:          i16       shard;
}

struct Unannotated {
  1: required i64 id;
}

@text.Format{"pattern": "{id}"}
struct OptionalField {
  1: optional i64 id;
}

@text.Format{"pattern": "{id}"}
union Union {
  1: i64 id;
}

@text.Format{"pattern": "{type}_{entity_id}"}
struct SeparatorConflict {
  1: required Type type;
  2: required i64  entity_id;
}
