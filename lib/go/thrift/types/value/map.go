package value

func EncodeMapValue(vs map[interface{}]interface{}) (*MapValue, error) {
	var es = make([]*MapEntry, 0, len(vs))

	for k, v := range vs {
		kv, err := EncodeValue(k)

		if err != nil {
			return nil, err
		}

		vv, err := EncodeValue(v)

		if err != nil {
			return nil, err
		}

		es = append(es, &MapEntry{Key: kv, Value: vv})
	}

	return &MapValue{Entries: es}, nil
}

func (mv *MapValue) ToMap() map[interface{}]interface{} {
	vs := make(map[interface{}]interface{}, len(mv.Entries))

	for _, e := range mv.Entries {
		vs[e.Key.ToInterface()] = e.Value.ToInterface()
	}

	return vs
}
