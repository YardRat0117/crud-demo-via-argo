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
			Entrypoint: "main",
			Templates: []wf.Template{
				{
					Name: "main",
					Steps: []wf.ParallelSteps{
						{
							Steps: []wf.WorkflowStep{
								{
									Name:     "validate-params",
									Template: "validate",
								},
							},
						},
					},
				},
				{
					Name: "validate",
					Container: &corev1.Container{
						Image: "my-validate:latest",
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
