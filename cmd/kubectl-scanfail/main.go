package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/0xNiel/k8s-app-plugin/pkg/scanner"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig string
	var namespace string
	var allNamespaces bool

	// Setup flags
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.StringVar(&namespace, "namespace", "", "namespace to scan (empty means all namespaces)")
	flag.BoolVar(&allNamespaces, "all-namespaces", false, "scan all namespaces")
	flag.BoolVar(&allNamespaces, "A", false, "scan all namespaces (shorthand)")

	flag.Parse()

	// Handle all-namespaces flag
	if allNamespaces {
		namespace = ""
	}

	// Create scanner
	s, err := scanner.NewScanner(kubeconfig, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating scanner: %v\n", err)
		os.Exit(1)
	}

	// Run scan
	ctx := context.Background()
	failures, err := s.ScanAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning cluster: %v\n", err)
		os.Exit(1)
	}

	// Print results
	if len(failures) == 0 {
		fmt.Println("✓ No failing resources found!")
		return
	}

	fmt.Printf("Found %d failing resource(s):\n\n", len(failures))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "KIND\tNAMESPACE\tNAME\tREASON\tDETAILS")
	fmt.Fprintln(w, "----\t---------\t----\t------\t-------")

	for _, failure := range failures {
		ns := failure.Namespace
		if ns == "" {
			ns = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			failure.Kind,
			ns,
			failure.Name,
			failure.Reason,
			failure.Details)
	}

	w.Flush()
	os.Exit(1) // Exit with error code when failures are found
}
