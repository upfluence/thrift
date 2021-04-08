package value

func EncodeListValue(vs []interface{}) (*ListValue, error) {
	var (
		err error

		tvs = make([]*Value, len(vs))
	)

	for i, v := range vs {
		tvs[i], err = EncodeValue(v)

		if err != nil {
			return nil, err
		}
	}

	return &ListValue{Values: tvs}, nil
}

func (lv *ListValue) ToSlice() []interface{} {
	vs := make([]interface{}, len(lv.Values))

	for i, v := range lv.Values {
		vs[i] = v.ToInterface()
	}

	return vs
}
