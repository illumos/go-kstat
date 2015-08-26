//
// This is a simple hacked together program to exercise the kstats
// package. The statistics it inspects and fiddles around with are not
// necessarily universal.
//
package main

import (
	"fmt"
	"log"

	"github.com/siebenmann/go-kstat"
)

//
// -------------------

// Find kstats that are not named or IO stats
// Currently the only known ones are type 0 for eg
// 	unix:0:page_retire_list misc, unix:0:kstat_headers kstat,
//	cpu_stat:2:cpu_stat2 misc
func findNonNamed(tok *kstat.Token) {
	for _, r := range tok.All() {
		if !(r.Type == 1 || r.Type == 3) {
			fmt.Printf("found: %s type=%d\n", r, r.Type)
		}
	}
}

func printStat(r *kstat.Named) {
	fmt.Printf("%-30s %-6s value ", r, r.Type)
	switch r.Type {
	case kstat.String, kstat.CharData:
		fmt.Printf("'%s'\n", r.StringVal)
	case kstat.Int32, kstat.Int64:
		fmt.Printf("%16d\n", r.IntVal)
	case kstat.Uint32, kstat.Uint64:
		fmt.Printf("%16d\n", r.UintVal)
	default:
		fmt.Printf("UNKNOWN\n")
	}
}

func reporton(tok *kstat.Token, module string, instance int, name, stat string) {
	r, err := tok.GetNamed(module, instance, name, stat)
	if err != nil {
		log.Printf("reporton: '%s:%d:%s:%s' error: %s\n", module, instance, name, stat, err)
		return
	}
	printStat(r)
}

func allNamed(ks *kstat.KStat) {
	lst, err := ks.AllNamed()
	if err != nil {
		log.Fatal("AllNamed error: ", err)
	}

	for _, st := range lst {
		printStat(st)
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("kstat-test: ")

	tok, err := kstat.Open()
	if err != nil {
	}
	t, err := tok.Lookup("", -1, "e1000g1")
	if err != nil {
		log.Fatal("entry lookup error: ", err)
	}
	r, err := t.GetNamed("obytes64")
	if err != nil {
		log.Fatal("stat lookup error: ", err)
	}
	fmt.Printf("kstat %s: %s type %s value %d\n", t, r.Name, r.Type, r.UintVal)
	r, err = tok.GetNamed("link", -1, "e1000g1", "opackets64")
	if err != nil {
		log.Fatal("stat 2 lookup error: ", err)
	}
	fmt.Printf("kstat %s value %d\n\n", r, r.UintVal)

	reporton(tok, "cpu_info", -1, "cpu_info0", "implementation")
	reporton(tok, "cpu_info", -1, "cpu_info0", "supported_frequencies_Hz")
	reporton(tok, "cpu_info", -1, "cpu_info0", "cpu_type")
	fmt.Println(" ")

	findNonNamed(tok)

	// Should be an error:
	fmt.Println(" ")
	reporton(tok, "sd", 0, "sd0", "random")
	reporton(tok, "fred", -1, "james", "bad")
	reporton(tok, "cpu_info", -1, "cpu_info0", "nosuch")

	fmt.Println("")
	t, err = tok.Lookup("cpu_info", -1, "cpu_info0")
	if err != nil {
		log.Fatal("cpu_info lookup error: ", err)
	}
	allNamed(t)
	fmt.Println("")

	err = tok.Close()
	if err != nil {
		log.Fatal("error on token close: ", err)
	}

	// check the defenses:
	reporton(nil, "fred", -1, "james", "bad")
	r, err = t.GetNamed("ibytes64")
	if err != nil {
		log.Print("got error: ", err)
	}
}
