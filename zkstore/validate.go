package zkstore

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	validNameRE     = regexp.MustCompile(`^[\w\d_-]*$`)
	validCategoryRE = regexp.MustCompile(`^[/\w\d_-]+$`)
)

// validateNamed validates items that have a "name". Like an actual Name
// or perhaps a Version.  Since some names can be blank, we use the
// 'required' parameter to signify whether or not a name can be blank, and
// then after that we check against the regexp.
func validateNamed(name string, required bool) error {
	if strings.TrimSpace(name) != name {
		return errors.New("leader or trailing spaces not allowed")
	}
	if required && name == "" {
		return errors.New("cannot be blank")
	}
	if !validNameRE.MatchString(name) {
		return fmt.Errorf("must match %s", validNameRE)
	}
	return nil
}

// a category is required, and can look like a path or not
func validateCategory(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("cannot be blank")
	}
	if !validCategoryRE.MatchString(name) {
		return fmt.Errorf("must match %s", validCategoryRE)
	}
	return nil
}
