package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/0xNiel/k8s-app-plugin/pkg/scanner"
	"k8s.io/client-go/util/homedir"
)

const version = "1.0.0"

func main() {
	var kubeconfig string
	var namespace string
	var allNamespaces bool
	var showVersion bool
	var outputFormat string

	// Setup flags
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.StringVar(&namespace, "namespace", "", "namespace to scan (empty means all namespaces)")
	flag.StringVar(&namespace, "n", "", "namespace to scan (shorthand)")
	flag.BoolVar(&allNamespaces, "all-namespaces", false, "scan all namespaces")
	flag.BoolVar(&allNamespaces, "A", false, "scan all namespaces (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&showVersion, "v", false, "show version information (shorthand)")
	flag.StringVar(&outputFormat, "output", "table", "output format: table, simple, or json")
	flag.StringVar(&outputFormat, "o", "table", "output format (shorthand)")

	flag.Parse()

	// Show version
	if showVersion {
		fmt.Printf("kscan version %s\n", version)
		return
	}

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

	// Print results based on format
	switch strings.ToLower(outputFormat) {
	case "table":
		printTable(failures)
	case "simple":
		printSimple(failures)
	case "json":
		printJSON(failures)
	default:
		fmt.Fprintf(os.Stderr, "Unknown output format: %s\n", outputFormat)
		os.Exit(1)
	}

	if len(failures) > 0 {
		os.Exit(1) // Exit with error code when failures are found
	}
}

func printTable(failures []scanner.FailingResource) {
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
}

func printSimple(failures []scanner.FailingResource) {
	if len(failures) == 0 {
		fmt.Println("No failing resources found")
		return
	}

	for _, failure := range failures {
		if failure.Namespace != "" {
			fmt.Printf("%s/%s/%s: %s - %s\n",
				failure.Kind,
				failure.Namespace,
				failure.Name,
				failure.Reason,
				failure.Details)
		} else {
			fmt.Printf("%s/%s: %s - %s\n",
				failure.Kind,
				failure.Name,
				failure.Reason,
				failure.Details)
		}
	}
}

func printJSON(failures []scanner.FailingResource) {
	if len(failures) == 0 {
		fmt.Println("[]")
		return
	}

	fmt.Println("[")
	for i, failure := range failures {
		fmt.Printf("  {\n")
		fmt.Printf("    \"kind\": \"%s\",\n", failure.Kind)
		fmt.Printf("    \"namespace\": \"%s\",\n", failure.Namespace)
		fmt.Printf("    \"name\": \"%s\",\n", failure.Name)
		fmt.Printf("    \"reason\": \"%s\",\n", failure.Reason)
		fmt.Printf("    \"details\": \"%s\",\n", failure.Details)
		fmt.Printf("    \"detectedAt\": \"%s\"\n", failure.DetectedAt.Format("2006-01-02T15:04:05Z07:00"))
		if i < len(failures)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}
