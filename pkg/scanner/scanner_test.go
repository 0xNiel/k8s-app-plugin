package scanner

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestScanPods(t *testing.T) {
	tests := []struct {
		name            string
		pods            []runtime.Object
		expectedCount   int
		expectedReasons []FailureReason
	}{
		{
			name: "healthy pod",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "healthy-pod",
						Namespace: "default",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name: "container-1",
								State: corev1.ContainerState{
									Running: &corev1.ContainerStateRunning{},
								},
							},
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "pod with failed phase",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "failed-pod",
						Namespace: "default",
					},
					Status: corev1.PodStatus{
						Phase:   corev1.PodFailed,
						Message: "Pod failed",
					},
				},
			},
			expectedCount:   1,
			expectedReasons: []FailureReason{PodPhaseFailed},
		},
		{
			name: "pod with container waiting",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "waiting-pod",
						Namespace: "default",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name: "container-1",
								State: corev1.ContainerState{
									Waiting: &corev1.ContainerStateWaiting{
										Reason:  "ImagePullBackOff",
										Message: "Cannot pull image",
									},
								},
							},
						},
					},
				},
			},
			expectedCount:   1,
			expectedReasons: []FailureReason{PodContainerWaiting},
		},
		{
			name: "pod running but not ready",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "not-ready-pod",
						Namespace: "default",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			expectedCount:   1,
			expectedReasons: []FailureReason{PodNotReady},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.pods...)
			s := &Scanner{
				clientset: clientset,
				namespace: "",
			}

			failures, err := s.ScanPods(context.Background())
			if err != nil {
				t.Fatalf("ScanPods() error = %v", err)
			}

			if len(failures) != tt.expectedCount {
				t.Errorf("Expected %d failures, got %d", tt.expectedCount, len(failures))
			}

			if tt.expectedReasons != nil {
				for i, expectedReason := range tt.expectedReasons {
					if i >= len(failures) {
						break
					}
					if failures[i].Reason != expectedReason {
						t.Errorf("Expected reason %s, got %s", expectedReason, failures[i].Reason)
					}
				}
			}
		})
	}
}

func TestScanDeployments(t *testing.T) {
	replicas := int32(3)
	tests := []struct {
		name          string
		deployments   []runtime.Object
		expectedCount int
	}{
		{
			name: "healthy deployment",
			deployments: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "healthy-deploy",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &replicas,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 3,
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "deployment with unavailable replicas",
			deployments: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "unhealthy-deploy",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &replicas,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.deployments...)
			s := &Scanner{
				clientset: clientset,
				namespace: "",
			}

			failures, err := s.ScanDeployments(context.Background())
			if err != nil {
				t.Fatalf("ScanDeployments() error = %v", err)
			}

			if len(failures) != tt.expectedCount {
				t.Errorf("Expected %d failures, got %d", tt.expectedCount, len(failures))
			}

			for _, failure := range failures {
				if failure.Reason != DeploymentUnavailable {
					t.Errorf("Expected reason %s, got %s", DeploymentUnavailable, failure.Reason)
				}
			}
		})
	}
}

func TestScanStatefulSets(t *testing.T) {
	replicas := int32(3)
	tests := []struct {
		name          string
		statefulsets  []runtime.Object
		expectedCount int
	}{
		{
			name: "healthy statefulset",
			statefulsets: []runtime.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "healthy-sts",
						Namespace: "default",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &replicas,
					},
					Status: appsv1.StatefulSetStatus{
						ReadyReplicas: 3,
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "statefulset with unready replicas",
			statefulsets: []runtime.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "unhealthy-sts",
						Namespace: "default",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &replicas,
					},
					Status: appsv1.StatefulSetStatus{
						ReadyReplicas: 1,
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.statefulsets...)
			s := &Scanner{
				clientset: clientset,
				namespace: "",
			}

			failures, err := s.ScanStatefulSets(context.Background())
			if err != nil {
				t.Fatalf("ScanStatefulSets() error = %v", err)
			}

			if len(failures) != tt.expectedCount {
				t.Errorf("Expected %d failures, got %d", tt.expectedCount, len(failures))
			}
		})
	}
}

func TestScanDaemonSets(t *testing.T) {
	tests := []struct {
		name          string
		daemonsets    []runtime.Object
		expectedCount int
	}{
		{
			name: "healthy daemonset",
			daemonsets: []runtime.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "healthy-ds",
						Namespace: "default",
					},
					Status: appsv1.DaemonSetStatus{
						NumberUnavailable:      0,
						DesiredNumberScheduled: 3,
						CurrentNumberScheduled: 3,
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "daemonset with unavailable pods",
			daemonsets: []runtime.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "unhealthy-ds",
						Namespace: "default",
					},
					Status: appsv1.DaemonSetStatus{
						NumberUnavailable:      2,
						DesiredNumberScheduled: 3,
						CurrentNumberScheduled: 3,
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.daemonsets...)
			s := &Scanner{
				clientset: clientset,
				namespace: "",
			}

			failures, err := s.ScanDaemonSets(context.Background())
			if err != nil {
				t.Fatalf("ScanDaemonSets() error = %v", err)
			}

			if len(failures) != tt.expectedCount {
				t.Errorf("Expected %d failures, got %d", tt.expectedCount, len(failures))
			}
		})
	}
}

func TestScanJobs(t *testing.T) {
	tests := []struct {
		name          string
		jobs          []runtime.Object
		expectedCount int
	}{
		{
			name: "successful job",
			jobs: []runtime.Object{
				&batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "successful-job",
						Namespace: "default",
					},
					Status: batchv1.JobStatus{
						Failed:    0,
						Succeeded: 1,
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "failed job",
			jobs: []runtime.Object{
				&batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "failed-job",
						Namespace: "default",
					},
					Status: batchv1.JobStatus{
						Failed:    3,
						Succeeded: 0,
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.jobs...)
			s := &Scanner{
				clientset: clientset,
				namespace: "",
			}

			failures, err := s.ScanJobs(context.Background())
			if err != nil {
				t.Fatalf("ScanJobs() error = %v", err)
			}

			if len(failures) != tt.expectedCount {
				t.Errorf("Expected %d failures, got %d", tt.expectedCount, len(failures))
			}
		})
	}
}

func TestScanNodes(t *testing.T) {
	tests := []struct {
		name          string
		nodes         []runtime.Object
		expectedCount int
	}{
		{
			name: "healthy node",
			nodes: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "healthy-node",
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "node not ready",
			nodes: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "unhealthy-node",
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:    corev1.NodeReady,
								Status:  corev1.ConditionFalse,
								Message: "Node is not ready",
							},
						},
					},
				},
			},
			expectedCount: 1,
		},
		{
			name: "node ready unknown",
			nodes: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "unknown-node",
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:    corev1.NodeReady,
								Status:  corev1.ConditionUnknown,
								Message: "Node status unknown",
							},
						},
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.nodes...)
			s := &Scanner{
				clientset: clientset,
				namespace: "",
			}

			failures, err := s.ScanNodes(context.Background())
			if err != nil {
				t.Fatalf("ScanNodes() error = %v", err)
			}

			if len(failures) != tt.expectedCount {
				t.Errorf("Expected %d failures, got %d", tt.expectedCount, len(failures))
			}
		})
	}
}

func TestScanPVCs(t *testing.T) {
	tests := []struct {
		name            string
		pvcs            []runtime.Object
		expectedCount   int
		expectedReasons []FailureReason
	}{
		{
			name: "bound pvc",
			pvcs: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bound-pvc",
						Namespace: "default",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "pending pvc",
			pvcs: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-pvc",
						Namespace: "default",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimPending,
					},
				},
			},
			expectedCount:   1,
			expectedReasons: []FailureReason{PVCPending},
		},
		{
			name: "lost pvc",
			pvcs: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "lost-pvc",
						Namespace: "default",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimLost,
					},
				},
			},
			expectedCount:   1,
			expectedReasons: []FailureReason{PVCLost},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.pvcs...)
			s := &Scanner{
				clientset: clientset,
				namespace: "",
			}

			failures, err := s.ScanPVCs(context.Background())
			if err != nil {
				t.Fatalf("ScanPVCs() error = %v", err)
			}

			if len(failures) != tt.expectedCount {
				t.Errorf("Expected %d failures, got %d", tt.expectedCount, len(failures))
			}

			if tt.expectedReasons != nil {
				for i, expectedReason := range tt.expectedReasons {
					if i >= len(failures) {
						break
					}
					if failures[i].Reason != expectedReason {
						t.Errorf("Expected reason %s, got %s", expectedReason, failures[i].Reason)
					}
				}
			}
		})
	}
}

func TestScanAll(t *testing.T) {
	replicas := int32(3)
	objects := []runtime.Object{
		// Failed pod
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failed-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		},
		// Unhealthy deployment
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "unhealthy-deploy",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
			},
		},
		// Pending PVC
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pending-pvc",
				Namespace: "default",
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Phase: corev1.ClaimPending,
			},
		},
	}

	clientset := fake.NewSimpleClientset(objects...)
	s := &Scanner{
		clientset: clientset,
		namespace: "",
	}

	failures, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll() error = %v", err)
	}

	// We expect at least 3 failures (one from each resource type above)
	if len(failures) < 3 {
		t.Errorf("Expected at least 3 failures, got %d", len(failures))
	}

	// Verify that failures are sorted
	for i := 1; i < len(failures); i++ {
		prev := failures[i-1]
		curr := failures[i]

		if prev.Kind > curr.Kind {
			t.Errorf("Failures not sorted by kind: %s > %s", prev.Kind, curr.Kind)
		}
		if prev.Kind == curr.Kind && prev.Namespace > curr.Namespace {
			t.Errorf("Failures not sorted by namespace: %s > %s", prev.Namespace, curr.Namespace)
		}
		if prev.Kind == curr.Kind && prev.Namespace == curr.Namespace && prev.Name > curr.Name {
			t.Errorf("Failures not sorted by name: %s > %s", prev.Name, curr.Name)
		}
	}
}

func TestFailingResource(t *testing.T) {
	fr := FailingResource{
		Kind:       "Pod",
		Namespace:  "default",
		Name:       "test-pod",
		Reason:     PodPhaseFailed,
		Details:    "Test details",
		DetectedAt: time.Now(),
	}

	if fr.Kind != "Pod" {
		t.Errorf("Expected Kind to be Pod, got %s", fr.Kind)
	}
	if fr.Namespace != "default" {
		t.Errorf("Expected Namespace to be default, got %s", fr.Namespace)
	}
	if fr.Reason != PodPhaseFailed {
		t.Errorf("Expected Reason to be PodPhaseFailed, got %s", fr.Reason)
	}
}
