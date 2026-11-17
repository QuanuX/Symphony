package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	repo := flag.String("repository", "", "clean selected source checkout")
	builds := flag.String("build-root", "", "six command candidate builds")
	qxctl := flag.String("qxctl-build-root", "", "reproducible qxctl candidate build")
	proof := flag.String("accordare-download", "", "actual canonical go mod download JSON")
	owner := flag.String("accordare-source-commit", "", "exact reviewed owner commit")
	output := flag.String("output", "", "new candidate closure evidence file")
	flag.Parse()
	if flag.NArg() != 0 || *repo == "" || *builds == "" || *qxctl == "" || *proof == "" || *owner == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "all candidate closure arguments are required")
		os.Exit(2)
	}
	outputPath, err := gcOutputPath(*repo, *output)
	if err != nil {
		fmt.Fprintln(os.Stderr, "candidate closure refused:", err)
		os.Exit(1)
	}
	result, err := gcVerify(*repo, *builds, *qxctl, *proof, *owner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "candidate closure refused:", err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		data = append(data, '\n')
		var f *os.File
		f, err = os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			_, err = f.Write(data)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Verified seven source-bound Go candidates; publication and final-host acceptance remain pending.")
}
