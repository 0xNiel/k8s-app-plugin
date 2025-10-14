package scanner

import (
	"context"
	"fmt"
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// FailureReason represents why a resource is considered failing
type FailureReason string

const (
	PodContainerWaiting    FailureReason = "ContainerWaiting"
	PodNotReady            FailureReason = "PodNotReady"
	PodPhaseFailed         FailureReason = "PhaseFailed"
	DeploymentUnavailable  FailureReason = "ReplicasUnavailable"
	StatefulSetUnavailable FailureReason = "ReplicasUnavailable"
	DaemonSetUnavailable   FailureReason = "PodsUnavailable"
	JobFailed              FailureReason = "JobFailed"
	CronJobLastJobFailed   FailureReason = "LastJobFailed"
	NodeNotReady           FailureReason = "NodeNotReady"
	PVCPending             FailureReason = "PVCPending"
	PVCLost                FailureReason = "PVCLost"
)

// FailingResource represents a resource that is failing
type FailingResource struct {
	Kind       string
	Namespace  string
	Name       string
	Reason     FailureReason
	Details    string
	DetectedAt time.Time
}

// Scanner scans a Kubernetes cluster for failing resources
type Scanner struct {
	clientset kubernetes.Interface
	namespace string // empty string means all namespaces
}

// NewScanner creates a new scanner instance
func NewScanner(kubeconfig string, namespace string) (*Scanner, error) {
	config, err := buildConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Scanner{
		clientset: clientset,
		namespace: namespace,
	}, nil
}

// buildConfig creates a Kubernetes config from kubeconfig path or in-cluster config
func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// Try in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to default kubeconfig location
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, configOverrides).ClientConfig()
	}
	return config, nil
}

// ScanAll scans all resource types and returns failing resources
func (s *Scanner) ScanAll(ctx context.Context) ([]FailingResource, error) {
	var allFailures []FailingResource

	// Scan each resource type
	failures, err := s.ScanPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan pods: %w", err)
	}
	allFailures = append(allFailures, failures...)

	failures, err = s.ScanDeployments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan deployments: %w", err)
	}
	allFailures = append(allFailures, failures...)

	failures, err = s.ScanStatefulSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan statefulsets: %w", err)
	}
	allFailures = append(allFailures, failures...)

	failures, err = s.ScanDaemonSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan daemonsets: %w", err)
	}
	allFailures = append(allFailures, failures...)

	failures, err = s.ScanJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan jobs: %w", err)
	}
	allFailures = append(allFailures, failures...)

	failures, err = s.ScanCronJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan cronjobs: %w", err)
	}
	allFailures = append(allFailures, failures...)

	failures, err = s.ScanNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan nodes: %w", err)
	}
	allFailures = append(allFailures, failures...)

	failures, err = s.ScanPVCs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan pvcs: %w", err)
	}
	allFailures = append(allFailures, failures...)

	// Sort by kind, namespace, name for consistent output
	sort.Slice(allFailures, func(i, j int) bool {
		if allFailures[i].Kind != allFailures[j].Kind {
			return allFailures[i].Kind < allFailures[j].Kind
		}
		if allFailures[i].Namespace != allFailures[j].Namespace {
			return allFailures[i].Namespace < allFailures[j].Namespace
		}
		return allFailures[i].Name < allFailures[j].Name
	})

	return allFailures, nil
}

// ScanPods scans for failing pods
func (s *Scanner) ScanPods(ctx context.Context) ([]FailingResource, error) {
	pods, err := s.clientset.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, pod := range pods.Items {
		// Check if pod phase is Failed
		if pod.Status.Phase == corev1.PodFailed {
			failures = append(failures, FailingResource{
				Kind:       "Pod",
				Namespace:  pod.Namespace,
				Name:       pod.Name,
				Reason:     PodPhaseFailed,
				Details:    fmt.Sprintf("Phase: %s, Message: %s", pod.Status.Phase, pod.Status.Message),
				DetectedAt: time.Now(),
			})
			continue
		}

		// Check for containers in Waiting state
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil {
				failures = append(failures, FailingResource{
					Kind:      "Pod",
					Namespace: pod.Namespace,
					Name:      pod.Name,
					Reason:    PodContainerWaiting,
					Details: fmt.Sprintf("Container %s: %s - %s",
						containerStatus.Name,
						containerStatus.State.Waiting.Reason,
						containerStatus.State.Waiting.Message),
					DetectedAt: time.Now(),
				})
			}
		}

		// Check if pod is not ready (and should be by now)
		// Skip if pod is still pending and just started
		if pod.Status.Phase == corev1.PodRunning {
			ready := false
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			if !ready {
				failures = append(failures, FailingResource{
					Kind:       "Pod",
					Namespace:  pod.Namespace,
					Name:       pod.Name,
					Reason:     PodNotReady,
					Details:    "Pod is running but not ready",
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return failures, nil
}

// ScanDeployments scans for failing deployments
func (s *Scanner) ScanDeployments(ctx context.Context) ([]FailingResource, error) {
	deployments, err := s.clientset.AppsV1().Deployments(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, deployment := range deployments.Items {
		if deployment.Spec.Replicas == nil {
			continue
		}
		desired := *deployment.Spec.Replicas
		available := deployment.Status.AvailableReplicas

		if available < desired {
			failures = append(failures, FailingResource{
				Kind:      "Deployment",
				Namespace: deployment.Namespace,
				Name:      deployment.Name,
				Reason:    DeploymentUnavailable,
				Details: fmt.Sprintf("Available replicas (%d) < Desired replicas (%d)",
					available, desired),
				DetectedAt: time.Now(),
			})
		}
	}

	return failures, nil
}

// ScanStatefulSets scans for failing statefulsets
func (s *Scanner) ScanStatefulSets(ctx context.Context) ([]FailingResource, error) {
	statefulsets, err := s.clientset.AppsV1().StatefulSets(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, sts := range statefulsets.Items {
		if sts.Spec.Replicas == nil {
			continue
		}
		desired := *sts.Spec.Replicas
		ready := sts.Status.ReadyReplicas

		if ready < desired {
			failures = append(failures, FailingResource{
				Kind:      "StatefulSet",
				Namespace: sts.Namespace,
				Name:      sts.Name,
				Reason:    StatefulSetUnavailable,
				Details: fmt.Sprintf("Ready replicas (%d) < Desired replicas (%d)",
					ready, desired),
				DetectedAt: time.Now(),
			})
		}
	}

	return failures, nil
}

// ScanDaemonSets scans for failing daemonsets
func (s *Scanner) ScanDaemonSets(ctx context.Context) ([]FailingResource, error) {
	daemonsets, err := s.clientset.AppsV1().DaemonSets(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, ds := range daemonsets.Items {
		if ds.Status.NumberUnavailable > 0 {
			failures = append(failures, FailingResource{
				Kind:      "DaemonSet",
				Namespace: ds.Namespace,
				Name:      ds.Name,
				Reason:    DaemonSetUnavailable,
				Details: fmt.Sprintf("%d pods unavailable (Desired: %d, Current: %d)",
					ds.Status.NumberUnavailable,
					ds.Status.DesiredNumberScheduled,
					ds.Status.CurrentNumberScheduled),
				DetectedAt: time.Now(),
			})
		}
	}

	return failures, nil
}

// ScanJobs scans for failing jobs
func (s *Scanner) ScanJobs(ctx context.Context) ([]FailingResource, error) {
	jobs, err := s.clientset.BatchV1().Jobs(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, job := range jobs.Items {
		if job.Status.Failed > 0 && job.Status.Succeeded == 0 {
			failures = append(failures, FailingResource{
				Kind:      "Job",
				Namespace: job.Namespace,
				Name:      job.Name,
				Reason:    JobFailed,
				Details: fmt.Sprintf("Failed: %d, Succeeded: %d",
					job.Status.Failed, job.Status.Succeeded),
				DetectedAt: time.Now(),
			})
		}
	}

	return failures, nil
}

// ScanCronJobs scans for failing cronjobs
func (s *Scanner) ScanCronJobs(ctx context.Context) ([]FailingResource, error) {
	cronjobs, err := s.clientset.BatchV1().CronJobs(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, cronjob := range cronjobs.Items {
		// Check if there's a last schedule time and if the most recent job failed
		if cronjob.Status.LastScheduleTime != nil && len(cronjob.Status.Active) == 0 {
			// Get the most recent job
			jobs, err := s.clientset.BatchV1().Jobs(cronjob.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name"),
			})
			if err != nil {
				continue
			}

			// Find jobs owned by this cronjob
			var ownedJobs []batchv1.Job
			for _, job := range jobs.Items {
				for _, owner := range job.OwnerReferences {
					if owner.Kind == "CronJob" && owner.Name == cronjob.Name {
						ownedJobs = append(ownedJobs, job)
						break
					}
				}
			}

			if len(ownedJobs) > 0 {
				// Sort by creation time to get the most recent
				sort.Slice(ownedJobs, func(i, j int) bool {
					return ownedJobs[i].CreationTimestamp.After(ownedJobs[j].CreationTimestamp.Time)
				})

				lastJob := ownedJobs[0]
				if lastJob.Status.Failed > 0 && lastJob.Status.Succeeded == 0 {
					failures = append(failures, FailingResource{
						Kind:      "CronJob",
						Namespace: cronjob.Namespace,
						Name:      cronjob.Name,
						Reason:    CronJobLastJobFailed,
						Details: fmt.Sprintf("Last job %s failed (Failed: %d)",
							lastJob.Name, lastJob.Status.Failed),
						DetectedAt: time.Now(),
					})
				}
			}
		}
	}

	return failures, nil
}

// ScanNodes scans for failing nodes
func (s *Scanner) ScanNodes(ctx context.Context) ([]FailingResource, error) {
	nodes, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown {
					failures = append(failures, FailingResource{
						Kind:       "Node",
						Namespace:  "", // Nodes are cluster-scoped
						Name:       node.Name,
						Reason:     NodeNotReady,
						Details:    fmt.Sprintf("Ready condition: %s - %s", condition.Status, condition.Message),
						DetectedAt: time.Now(),
					})
				}
				break
			}
		}
	}

	return failures, nil
}

// ScanPVCs scans for failing persistent volume claims
func (s *Scanner) ScanPVCs(ctx context.Context) ([]FailingResource, error) {
	pvcs, err := s.clientset.CoreV1().PersistentVolumeClaims(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var failures []FailingResource
	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase == corev1.ClaimPending {
			failures = append(failures, FailingResource{
				Kind:       "PersistentVolumeClaim",
				Namespace:  pvc.Namespace,
				Name:       pvc.Name,
				Reason:     PVCPending,
				Details:    "PVC is in Pending state",
				DetectedAt: time.Now(),
			})
		} else if pvc.Status.Phase == corev1.ClaimLost {
			failures = append(failures, FailingResource{
				Kind:       "PersistentVolumeClaim",
				Namespace:  pvc.Namespace,
				Name:       pvc.Name,
				Reason:     PVCLost,
				Details:    "PVC is in Lost state",
				DetectedAt: time.Now(),
			})
		}
	}

	return failures, nil
}
