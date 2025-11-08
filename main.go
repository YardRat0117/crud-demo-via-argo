package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflowclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	namespace := "argo"

	// Load kubeconfig
	kubeconfigPath := filepath.Join("./", "kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		panic(err)
	}

	// Kubernetes client
	k8sClient, err := k8sclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Argo workflow client
	wfClient, err := workflowclientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// 创建 PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio-pvc",
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	_, err = k8sClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		panic(err)
	}

	// 创建 MinIO Deployment
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "minio"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "minio"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "minio",
							Image: "quay.io/minio/minio:latest",
							Args:  []string{"server", "/data", "--console-address", ":9001"},
							Ports: []corev1.ContainerPort{
								{Name: "api", ContainerPort: 9000},
								{Name: "console", ContainerPort: 9001},
							},
							Env: []corev1.EnvVar{
								{Name: "MINIO_ROOT_USER", Value: "minioadmin"},
								{Name: "MINIO_ROOT_PASSWORD", Value: "thisisfortesting"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "minio-data", MountPath: "/data"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "minio-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "minio-pvc",
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = k8sClient.AppsV1().Deployments(namespace).Create(context.Background(), deploy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		panic(err)
	}

	// 3️⃣ 创建 Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "minio"},
			Ports: []corev1.ServicePort{
				{Name: "api", Port: 9000, TargetPort: intstr.FromInt(9000)},
				{Name: "console", Port: 9001, TargetPort: intstr.FromInt(9001)},
			},
		},
	}
	_, err = k8sClient.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		panic(err)
	}

	// 4️⃣ 等待 MinIO Pod Ready
	fmt.Println("Waiting for MinIO pod to be ready...")
	waitForMinioReady(k8sClient, namespace, "app=minio")

	// 5️⃣ 提交 workflow
	workflow := buildWorkflow(namespace)
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Create(context.Background(), workflow, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Workflow %s submitted successfully!\n", wf.Name)
}

// ---- helper functions ----

func int32Ptr(i int32) *int32 { return &i }

func waitForMinioReady(client *k8sclient.Clientset, namespace, labelSelector string) {
	for {
		pods, _ := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
		ready := false
		for _, pod := range pods.Items {
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ready = true
				}
			}
		}
		if ready {
			break
		}
		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}
	fmt.Println("\nMinIO pod is ready")
}

// Build Workflow object
func buildWorkflow(namespace string) *wfv1.Workflow {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "minio-crud-",
			Namespace:    namespace,
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint:         "main",
			ServiceAccountName: "argo",
			Templates: []wfv1.Template{
				{
					Name: "main",
					Steps: []wfv1.ParallelSteps{
						{
							Steps: []wfv1.WorkflowStep{
								{
									Name:     "run-create",
									Template: "create",
								},
							},
						},
						{
							Steps: []wfv1.WorkflowStep{
								{
									Name:     "run-read",
									Template: "read",
								},
							},
						},
						{
							Steps: []wfv1.WorkflowStep{
								{
									Name:     "run-update",
									Template: "update",
								},
							},
						},
						{
							Steps: []wfv1.WorkflowStep{
								{
									Name:     "run-delete",
									Template: "delete",
								},
							},
						},
					},
				},

				// Create
				buildTemplate("create", `
				FILE="/tmp/himinio.txt"
				BUCKET="testbucket"
				ENDPOINT="http://minio:9000"

				aws --endpoint-url $ENDPOINT s3 mb s3://$BUCKET || true
				echo "create success" >> $FILE
				aws --endpoint-url $ENDPOINT s3 cp $FILE s3://$BUCKET/himinio.txt
				cat $FILE
				`),

				// Read
				buildTemplate("read", `
				FILE="/tmp/himinio.txt"
				BUCKET="testbucket"
				ENDPOINT="http://minio:9000"

				aws --endpoint-url $ENDPOINT s3 cp s3://$BUCKET/himinio.txt $FILE || touch $FILE
				echo "read success" >> $FILE
				aws --endpoint-url $ENDPOINT s3 cp $FILE s3://$BUCKET/himinio.txt
				cat $FILE
				`),

				// Update
				buildTemplate("update", `
				FILE="/tmp/himinio.txt"
				BUCKET="testbucket"
				ENDPOINT="http://minio:9000"

				aws --endpoint-url $ENDPOINT s3 cp s3://$BUCKET/himinio.txt $FILE || touch $FILE
				echo "update success" >> $FILE
				aws --endpoint-url $ENDPOINT s3 cp $FILE s3://$BUCKET/himinio.txt
				cat $FILE
				`),

				// Delete
				buildTemplate("delete", `
				FILE="/tmp/himinio.txt"
				BUCKET="testbucket"
				ENDPOINT="http://minio:9000"

				aws --endpoint-url $ENDPOINT s3 cp s3://$BUCKET/himinio.txt $FILE || touch $FILE
				echo "delete success" >> $FILE
				aws --endpoint-url $ENDPOINT s3 cp $FILE s3://$BUCKET/himinio.txt
				cat $FILE
				`),
			},
		},
	}
}

func buildTemplate(name, script string) wfv1.Template {
	return wfv1.Template{
		Name: name,
		Container: &corev1.Container{
			Image: "amazon/aws-cli:latest",
			Env: []corev1.EnvVar{
				{Name: "AWS_ACCESS_KEY_ID", Value: "minioadmin"},
				{Name: "AWS_SECRET_ACCESS_KEY", Value: "thisisfortesting"},
			},
			Command: []string{"sh", "-c"},
			Args:    []string{script},
		},
	}
}

func homeDir() string {
	if h := filepath.Join("/root"); h != "" {
		return h
	}
	return "/"
}
