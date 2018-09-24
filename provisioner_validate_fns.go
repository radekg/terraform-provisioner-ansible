package main

import (
	"fmt"
	"strings"
)

func vfBecomeMethod(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if !becomeMethods[v] {
		errs = append(errs, fmt.Errorf("%s is not a valid become_method", v))
	}
	return
}

func vfPath(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.Index(v, "${path.module}") > -1 {
		warns = append(warns, fmt.Sprintf("I could not reliably determine the existence of '%s', most likely because of https://github.com/hashicorp/terraform/issues/17439. If the file does not exist, you'll experience a failure at runtime.", v))
	} else {
		if _, err := resolvePath(v); err != nil {
			errs = append(errs, fmt.Errorf("file '%s' does not exist", v))
		}
	}
	return
}
