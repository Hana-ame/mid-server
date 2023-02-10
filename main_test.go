package main

import (
	"fmt"
	"regexp"
	"testing"
)

func Test(t *testing.T) {
	re, err := regexp.Compile("inbox")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%t", re.Match([]byte("asdf/inbox/sf")))
}
