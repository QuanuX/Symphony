package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	plan := flag.String("plan", "", "reviewed acceptance plan")
	packet := flag.String("packet", "", "new template or existing evidence packet")
	output := flag.String("report", "", "new report file for check")
	mode := flag.String("mode", "check", "init or check")
	flag.Parse()
	fail := func() { fmt.Fprintln(os.Stderr, gateError); os.Exit(1) }
	if flag.NArg() != 0 || !filepath.IsAbs(*plan) || !filepath.IsAbs(*packet) {
		fail()
	}
	p, raw, e := loadGatePlan(*plan)
	if e != nil {
		fail()
	}
	if *mode == "init" {
		if *output != "" || writeNew(*packet, newPacket(p, raw)) != nil {
			fail()
		}
		fmt.Println("Prepared unqualified acceptance packet")
		return
	}
	if *mode != "check" || !filepath.IsAbs(*output) {
		fail()
	}
	data, e := bounded(*packet)
	if e != nil {
		fail()
	}
	result, e := checkPacket(p, raw, data, filepath.Dir(*packet))
	if e != nil {
		fail()
	}
	if writeNew(*output, result) != nil {
		fail()
	}
	if !result.EvidenceReady {
		fmt.Println("Acceptance inputs/evidence incomplete")
		os.Exit(2)
	}
	fmt.Println("Evidence packet ready for independent review; acceptance and publication remain unapproved")
}
