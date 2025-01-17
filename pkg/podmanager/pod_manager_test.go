// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package podmanager_test

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/spiderpool/pkg/constant"
	"github.com/spidernet-io/spiderpool/pkg/podmanager"
)

var _ = Describe("PodManager", Label("pod_manager_test"), func() {
	Describe("New PodManager", func() {
		It("sets default config", func() {
			manager, err := podmanager.NewPodManager(podmanager.PodManagerConfig{}, fakeClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(manager).NotTo(BeNil())
		})

		It("inputs nil client", func() {
			manager, err := podmanager.NewPodManager(podmanager.PodManagerConfig{}, nil)
			Expect(err).To(MatchError(constant.ErrMissingRequiredParam))
			Expect(manager).To(BeNil())
		})
	})

	Describe("Test PodManager's method", func() {
		var count uint64
		var namespace string
		var podName string
		var labels map[string]string
		var podT *corev1.Pod

		BeforeEach(func() {
			atomic.AddUint64(&count, 1)
			namespace = "default"
			podName = fmt.Sprintf("pod-%v", count)
			labels = map[string]string{"foo": fmt.Sprintf("bar-%v", count)}
			podT = &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{},
			}
		})

		var deleteOption *client.DeleteOptions

		AfterEach(func() {
			policy := metav1.DeletePropagationForeground
			deleteOption = &client.DeleteOptions{
				GracePeriodSeconds: pointer.Int64(0),
				PropagationPolicy:  &policy,
			}

			ctx := context.TODO()
			err := fakeClient.Delete(ctx, podT, deleteOption)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())
		})

		Describe("GetPodByName", func() {
			It("gets non-existent Pod", func() {
				ctx := context.TODO()
				pod, err := podManager.GetPodByName(ctx, namespace, podName)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
				Expect(pod).To(BeNil())
			})

			It("gets an existing Pod", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				pod, err := podManager.GetPodByName(ctx, namespace, podName)
				Expect(err).NotTo(HaveOccurred())
				Expect(pod).NotTo(BeNil())

				Expect(pod).To(Equal(podT))
			})
		})

		Describe("ListPods", func() {
			It("failed to list Pods due to some unknown errors", func() {
				patches := gomonkey.ApplyMethodReturn(fakeClient, "List", constant.ErrUnknown)
				defer patches.Reset()

				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				podList, err := podManager.ListPods(ctx)
				Expect(err).To(MatchError(constant.ErrUnknown))
				Expect(podList).To(BeNil())
			})

			It("lists all Pods", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				podList, err := podManager.ListPods(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(podList.Items).NotTo(BeEmpty())

				hasPod := false
				for _, pod := range podList.Items {
					if pod.Name == podName {
						hasPod = true
						break
					}
				}
				Expect(hasPod).To(BeTrue())
			})

			It("filters results by Namespace", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				podList, err := podManager.ListPods(ctx, client.InNamespace(namespace))
				Expect(err).NotTo(HaveOccurred())
				Expect(podList.Items).NotTo(BeEmpty())

				hasPod := false
				for _, pod := range podList.Items {
					if pod.Name == podName {
						hasPod = true
						break
					}
				}
				Expect(hasPod).To(BeTrue())
			})

			It("filters results by label selector", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				podList, err := podManager.ListPods(ctx, client.MatchingLabels(labels))
				Expect(err).NotTo(HaveOccurred())
				Expect(podList.Items).NotTo(BeEmpty())

				hasPod := false
				for _, pod := range podList.Items {
					if pod.Name == podName {
						hasPod = true
						break
					}
				}
				Expect(hasPod).To(BeTrue())
			})

			It("filters results by field selector", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				podList, err := podManager.ListPods(ctx, client.MatchingFields{metav1.ObjectNameField: podName})
				Expect(err).NotTo(HaveOccurred())
				Expect(podList.Items).NotTo(BeEmpty())

				hasPod := false
				for _, pod := range podList.Items {
					if pod.Name == podName {
						hasPod = true
						break
					}
				}
				Expect(hasPod).To(BeTrue())
			})
		})

		Describe("MatchLabelSelector", func() {
			It("checks non-existent Pod", func() {
				ctx := context.TODO()
				match, err := podManager.MatchLabelSelector(ctx, namespace, podName, &metav1.LabelSelector{MatchLabels: labels})
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(BeFalse())
			})

			It("matches invalid label selector", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				invalidSelector := &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"": "",
					},
				}
				match, err := podManager.MatchLabelSelector(ctx, namespace, podName, invalidSelector)
				Expect(err).To(HaveOccurred())
				Expect(match).To(BeFalse())
			})

			It("failed to list Pods due to some unknown errors", func() {
				patches := gomonkey.ApplyMethodReturn(fakeClient, "List", constant.ErrUnknown)
				defer patches.Reset()

				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				match, err := podManager.MatchLabelSelector(ctx, namespace, podName, &metav1.LabelSelector{MatchLabels: labels})
				Expect(err).To(MatchError(constant.ErrUnknown))
				Expect(match).To(BeFalse())
			})

			It("matches nothing", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				match, err := podManager.MatchLabelSelector(ctx, namespace, podName, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(BeFalse())
			})

			It("matches the label selector", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				match, err := podManager.MatchLabelSelector(ctx, namespace, podName, &metav1.LabelSelector{MatchLabels: labels})
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(BeTrue())
			})
		})

		Describe("MergeAnnotations", func() {
			It("merges annotations to non-existent Pod", func() {
				ctx := context.TODO()
				anno := map[string]string{"foo": "bar"}
				err := podManager.MergeAnnotations(ctx, namespace, podName, anno)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			It("merges empty annotations", func() {
				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				err = podManager.MergeAnnotations(ctx, namespace, podName, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("failed to update Pod due to some unknown errors", func() {
				patches := gomonkey.ApplyMethodReturn(fakeClient, "Update", constant.ErrUnknown)
				defer patches.Reset()

				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				anno := map[string]string{"foo": "bar"}
				err = podManager.MergeAnnotations(ctx, namespace, podName, anno)
				Expect(err).To(MatchError(constant.ErrUnknown))
			})

			It("runs out of retries to update Pod, but conflicts still occur", func() {
				patches := gomonkey.ApplyMethodReturn(fakeClient, "Update", apierrors.NewConflict(schema.GroupResource{Resource: "test"}, "other", nil))
				defer patches.Reset()

				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				anno := map[string]string{"foo": "bar"}
				err = podManager.MergeAnnotations(ctx, namespace, podName, anno)
				Expect(err).To(MatchError(constant.ErrRetriesExhausted))
			})

			It("merges annotations to Pod", func() {
				podT.SetAnnotations(map[string]string{
					"foo":   "merge",
					"exist": "value",
				})

				ctx := context.TODO()
				err := fakeClient.Create(ctx, podT)
				Expect(err).NotTo(HaveOccurred())

				anno := map[string]string{
					"foo": "bar",
					"new": "value",
				}
				err = podManager.MergeAnnotations(ctx, namespace, podName, anno)
				Expect(err).NotTo(HaveOccurred())

				var pod corev1.Pod
				err = fakeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: podName}, &pod)
				Expect(err).NotTo(HaveOccurred())
				Expect(pod.GetAnnotations()).To(Equal(
					map[string]string{
						"foo":   "bar",
						"exist": "value",
						"new":   "value",
					},
				))
			})
		})

		PDescribe("GetPodTopController", func() {})
	})
})
