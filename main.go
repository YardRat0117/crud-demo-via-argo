package main

import (
	"context"
	"fmt"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Read kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", "kubeconfig")
	if err != nil {
		panic(err)
	}

	// Argo workflow client
	argoClient, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create workflow
	workflow := &wf.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "echo-demo-task-",
			Namespace:    "argo",
		},
		Spec: wf.WorkflowSpec{
			Entrypoint:         "main",
			ServiceAccountName: "argo",
			Templates: []wf.Template{
				{
					Name: "main",
					Steps: []wf.ParallelSteps{
						{
							Steps: []wf.WorkflowStep{
								{Name: "run-create", Template: "create"},
							},
						},
						{
							Steps: []wf.WorkflowStep{
								{Name: "run-read", Template: "read"},
							},
						},
						{
							Steps: []wf.WorkflowStep{
								{Name: "run-update", Template: "update"},
							},
						},
						{
							Steps: []wf.WorkflowStep{
								{Name: "run-delete", Template: "delete"},
							},
						},
					},
				},
				{
					Name: "create",
					Container: &corev1.Container{
						Image:           "my-create:latest",
						ImagePullPolicy: "IfNotPresent",
						Command:         []string{"sh", "-c"},
						Args:            []string{"echo true > /tmp/created.txt"},
					},
					Outputs: wf.Outputs{
						Parameters: []wf.Parameter{
							{
								Name: "created",
								ValueFrom: &wf.ValueFrom{
									Path: "/tmp/created.txt",
								},
							},
						},
					},
				},
				{
					Name: "read",
					Container: &corev1.Container{
						Image:           "my-read:latest",
						ImagePullPolicy: "IfNotPresent",
						Command:         []string{"sh", "-c"},
						Args:            []string{"echo true > /tmp/read.txt"},
					},
					Outputs: wf.Outputs{
						Parameters: []wf.Parameter{
							{
								Name: "read",
								ValueFrom: &wf.ValueFrom{
									Path: "/tmp/read.txt",
								},
							},
						},
					},
				},
				{
					Name: "update",
					Container: &corev1.Container{
						Image:           "my-update:latest",
						ImagePullPolicy: "IfNotPresent",
						Command:         []string{"sh", "-c"},
						Args:            []string{"echo true > /tmp/updated.txt"},
					},
					Outputs: wf.Outputs{
						Parameters: []wf.Parameter{
							{
								Name: "updated",
								ValueFrom: &wf.ValueFrom{
									Path: "/tmp/updated.txt",
								},
							},
						},
					},
				},
				{
					Name: "delete",
					Container: &corev1.Container{
						Image:           "my-delete:latest",
						ImagePullPolicy: "IfNotPresent",
						Command:         []string{"sh", "-c"},
						Args:            []string{"echo true > /tmp/deleted.txt"},
					},
					Outputs: wf.Outputs{
						Parameters: []wf.Parameter{
							{
								Name: "deleted",
								ValueFrom: &wf.ValueFrom{
									Path: "/tmp/deleted.txt",
								},
							},
						},
					},
				},
			},
		},
	}

	// submit workflow
	wfClient := argoClient.ArgoprojV1alpha1().Workflows("argo")
	wfResp, err := wfClient.Create(context.Background(), workflow, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Workflow submitted:", wfResp.Name)
}
