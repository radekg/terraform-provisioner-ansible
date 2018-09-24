package main

func mapFromSetList(i []interface{}) map[string]interface{} {
	for _, v := range i {
		return v.(map[string]interface{})
	}
	return make(map[string]interface{})
}
