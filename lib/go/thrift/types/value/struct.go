package value

func EncodeStructValue(vs map[string]interface{}) (*StructValue, error) {
	var fs = make(map[string]*Value, len(vs))

	for k, v := range vs {
		tv, err := EncodeValue(v)

		if err != nil {
			return nil, err
		}

		fs[k] = tv
	}

	return &StructValue{Fields: fs}, nil
}

func (sv *StructValue) ToMap() map[string]interface{} {
	vs := make(map[string]interface{}, len(sv.Fields))

	for k, f := range sv.Fields {
		vs[k] = f.ToInterface()
	}

	return vs
}
