package main

import (
	"fmt"
	"testing"

	"github.com/tekintian/go-bbdown/util"
)

func TestBVConverter(t *testing.T) {
	converter := util.NewBVConverter()
	av, err := converter.BVToAV("BV1NdmuBXEWe")
	if err != nil {
		t.Error("Error:", err)
	} else {
		fmt.Println("AV:", av)
	}
}
