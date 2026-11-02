package main

import (
	"reflect"
	"testing"
)

func TestTypedResultComparisonPreservesLargeIntegers(t *testing.T) {
	a := decode([]byte(`{"integer":9007199254740993,"rational":{"numerator":"7","denominator":"13"},"unavailable":null}`))
	b := decode([]byte(`{"integer":9007199254740992,"rational":{"numerator":"7","denominator":"13"},"unavailable":null}`))
	if reflect.DeepEqual(a, b) {
		t.Fatal("rounded authoritative integer")
	}
	if !reflect.DeepEqual(a, copyObject(a)) {
		t.Fatal("copy changed typed result")
	}
}
func TestJobRequestDoesNotMutateAddress(t *testing.T) {
	a := object{"directory": "/private/job", "expected_generation": "0"}
	r := jobRequest("job_reconcile", a)
	if len(a) != 2 || r["protocol"] != "symphony.sbv.job-reconcile-input.v1" {
		t.Fatal("request alias")
	}
}
