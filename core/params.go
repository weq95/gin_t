package core

import "net/url"

type Param struct {
	Key string
	Val string
}

type Params []Param

func (p Params) byName(name string) string {
	for idx, _ := range p {
		if p[idx].Key == name {
			return p[idx].Val
		}
	}

	return ""
}

func (p Params) Get(key string) Value {
	return Value(p.byName(key))
}

func (p Params) ToURLValues() url.Values {
	var values = url.Values{}
	for _, val := range p {
		values.Add(val.Key, val.Val)
	}

	return values
}
