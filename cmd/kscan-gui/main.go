package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/odnielgonzalez/k8s-app-plugin/pkg/scanner"
	"k8s.io/client-go/util/homedir"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("KScan - Kubernetes Failure Scanner")
	myWindow.Resize(fyne.NewSize(900, 600))

	// Create UI components
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	kubeconfigEntry := widget.NewEntry()
	kubeconfigEntry.SetText(kubeconfigPath)
	kubeconfigEntry.SetPlaceHolder("Path to kubeconfig file")

	namespaceEntry := widget.NewEntry()
	namespaceEntry.SetPlaceHolder("Namespace (empty for all)")

	statusLabel := widget.NewLabel("Ready to scan")
	scanButton := widget.NewButton("Scan Cluster", func() {})

	// Results list
	resultsData := binding.NewStringList()
	resultsList := widget.NewListWithData(
		resultsData,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			strItem := item.(binding.String)
			val, _ := strItem.Get()
			label.SetText(val)
		},
	)

	// Auto-refresh checkbox and interval
	autoRefresh := widget.NewCheck("Auto-refresh", func(checked bool) {})
	refreshInterval := widget.NewSelect([]string{"10s", "30s", "1m", "5m"}, func(value string) {})
	refreshInterval.SetSelected("30s")

	var stopRefresh chan bool
	var scanning bool

	// Scan function
	doScan := func() {
		if scanning {
			return
		}
		scanning = true
		scanButton.Disable()
		statusLabel.SetText("Scanning cluster...")

		go func() {
			kubeconfig := kubeconfigEntry.Text
			namespace := namespaceEntry.Text

			s, err := scanner.NewScanner(kubeconfig, namespace)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				scanButton.Enable()
				scanning = false
				return
			}

			ctx := context.Background()
			failures, err := s.ScanAll(ctx)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				scanButton.Enable()
				scanning = false
				return
			}

			// Update results
			var results []string
			if len(failures) == 0 {
				results = append(results, "✓ No failing resources found!")
			} else {
				results = append(results, fmt.Sprintf("Found %d failing resource(s):", len(failures)))
				results = append(results, "")
				for _, failure := range failures {
					ns := failure.Namespace
					if ns == "" {
						ns = "cluster-scoped"
					}
					results = append(results, fmt.Sprintf("→ %s/%s/%s",
						failure.Kind, ns, failure.Name))
					results = append(results, fmt.Sprintf("  Reason: %s", failure.Reason))
					results = append(results, fmt.Sprintf("  Details: %s", failure.Details))
					results = append(results, "")
				}
			}

			resultsData.Set(results)
			statusLabel.SetText(fmt.Sprintf("Last scan: %s - Found %d failing resources",
				time.Now().Format("15:04:05"), len(failures)))
			scanButton.Enable()
			scanning = false
		}()
	}

	// Set up scan button action
	scanButton.OnTapped = doScan

	// Auto-refresh logic
	autoRefresh.OnChanged = func(checked bool) {
		if checked {
			stopRefresh = make(chan bool)
			go func() {
				intervalStr := refreshInterval.Selected
				var duration time.Duration
				switch intervalStr {
				case "10s":
					duration = 10 * time.Second
				case "30s":
					duration = 30 * time.Second
				case "1m":
					duration = 1 * time.Minute
				case "5m":
					duration = 5 * time.Minute
				default:
					duration = 30 * time.Second
				}

				ticker := time.NewTicker(duration)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						doScan()
					case <-stopRefresh:
						return
					}
				}
			}()
		} else {
			if stopRefresh != nil {
				close(stopRefresh)
			}
		}
	}

	// Layout
	configForm := container.New(layout.NewFormLayout(),
		widget.NewLabel("Kubeconfig:"),
		kubeconfigEntry,
		widget.NewLabel("Namespace:"),
		namespaceEntry,
	)

	refreshContainer := container.NewHBox(
		autoRefresh,
		widget.NewLabel("Interval:"),
		refreshInterval,
	)

	controls := container.NewVBox(
		configForm,
		container.NewHBox(
			scanButton,
			layout.NewSpacer(),
			refreshContainer,
		),
		widget.NewSeparator(),
		statusLabel,
		widget.NewSeparator(),
	)

	resultsContainer := container.NewBorder(
		widget.NewLabel("Results:"),
		nil, nil, nil,
		container.NewScroll(resultsList),
	)

	content := container.NewBorder(
		controls,
		nil, nil, nil,
		resultsContainer,
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}
