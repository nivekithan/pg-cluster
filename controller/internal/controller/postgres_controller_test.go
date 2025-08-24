/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

var _ = Describe("Postgres Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		postgres := &databasev1.Postgres{}

		BeforeEach(func() {
			By("creating the test secret")
			secretName := "test-postgres-secret"
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"POSTGRES_PASSWORD":    []byte("testpass"),
					"POSTGRES_USER":        []byte("testuser"),
					"POSTGRES_DB":          []byte("testdb"),
					"PGDATA":               []byte("/var/lib/postgresql/data"),
					"S3_BUCKET_NAME":       []byte("test-bucket"),
					"S3_ENDPOINT":          []byte("minio:9000"),
					"S3_ACCESS_KEY":        []byte("testkey"),
					"S3_ACCESS_KEY_SECRET": []byte("testsecret"),
					"S3_REGION":            []byte("us-east-1"),
					"REPO1_RETENTION_FULL": []byte("7"),
					"STANZA_NAME":          []byte("test-stanza"),
					"ARCHIVE_TIMEOUT":      []byte("60"),
					"MAX_WAL_SENDERS":      []byte("3"),
					"WAL_KEEP_SIZE":        []byte("1024"),
					"PG1_SOCKET_PATH":      []byte("/tmp/postgres"),
				},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, &corev1.Secret{})
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			}

			By("creating the custom resource for the Kind Postgres")
			err = k8sClient.Get(ctx, typeNamespacedName, postgres)
			if err != nil && errors.IsNotFound(err) {
				resource := &databasev1.Postgres{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: databasev1.PostgresSpec{
						Size:              resource.MustParse("1Gi"),
						DatabaseList:      []string{"testdb1", "testdb2"},
						PostgresSecretRef: secretName,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance Postgres")
			resource := &databasev1.Postgres{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			By("Cleanup the test secret")
			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-postgres-secret", Namespace: "default"}, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PostgresReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
