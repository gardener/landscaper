package terraformer

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	kutils "github.com/gardener/landscaper/pkg/utils/kubernetes"
	mock_client "github.com/gardener/landscaper/pkg/utils/kubernetes/mock"
)

var (
	configMapGroupResource      = schema.GroupResource{Resource: "ConfigMaps"}
	secretGroupResource         = schema.GroupResource{Resource: "Secrets"}
	serviceAccountGroupResource = schema.GroupResource{Resource: "ServiceAccounts"}
	roleGroupResource           = schema.GroupResource{Resource: "Roles"}
	roleBindingGroupResource    = schema.GroupResource{Resource: "RoleBindings"}
	podGroupResource            = schema.GroupResource{Resource: "Pods"}
)

var _ = Describe("terraformer", func() {
	const (
		namespace      = "namespace"
		logLevel       = "info"
		itemNamespace  = "itemNamespace"
		itemName       = "itemName"
		itemGeneration = int64(1)

		main      = "main"
		variables = "variables"
		tfvars    = "tfvars"

		// 3d0979630abd is the expected 12 characters id
		// generated from the sha256sum of 'itemNamespace/itemName'
		// echo -n "itemNamespace/itemName"| sha256sum | head -c12
		id                         = "3d0979630abd"
		expectedConfigurationName  = id + ".tf-config"
		expectedTFVarsName         = id + ".tf-vars"
		expectedStateName          = id + ".tf-state"
		expectedTerraformerName    = "terraformer-" + id
		expectedPodName            = expectedTerraformerName
		expectedServiceAccountName = expectedTerraformerName
		expectedRoleName           = expectedTerraformerName
		expectedRoleBindingName    = expectedTerraformerName
	)

	var (
		terraformContainer = terraformv1alpha1.ContainerSpec{
			Image:           "image:tag",
			ImagePullPolicy: corev1.PullIfNotPresent,
		}
		initContainer = terraformv1alpha1.ContainerSpec{
			Image:           "image:tag",
			ImagePullPolicy: corev1.PullIfNotPresent,
		}
	)
	var (
		ctrl       *gomock.Controller
		fakeClient *mock_client.MockClient
		ctx        context.Context

		tfr *Terraformer
		log logr.Logger
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		fakeClient = mock_client.NewMockClient(ctrl)

		ctx = context.Background()

		log = logzap.New(logzap.WriteTo(GinkgoWriter))

		var restConfig *rest.Config
		tfr = New(log, fakeClient, restConfig, namespace, logLevel, itemNamespace, itemName, initContainer, terraformContainer)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#Resources", func() {
		var (
			configurationKey client.ObjectKey
			tfvarsKey        client.ObjectKey
			stateKey         client.ObjectKey

			configurationObjectMeta metav1.ObjectMeta
			tfvarsObjectMeta        metav1.ObjectMeta
			stateObjectMeta         metav1.ObjectMeta

			expectedConfiguration *corev1.ConfigMap
			expectedInitState     *corev1.ConfigMap
			expectedTFVars        *corev1.Secret

			labels map[string]string
		)
		BeforeEach(func() {
			labels = map[string]string{
				LabelKeyItemName:      itemName,
				LabelKeyItemNamespace: itemNamespace,
			}

			configurationKey = kutils.ObjectKey(expectedConfigurationName, namespace)
			tfvarsKey = kutils.ObjectKey(expectedTFVarsName, namespace)
			stateKey = kutils.ObjectKey(expectedStateName, namespace)

			configurationObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: expectedConfigurationName, Labels: labels}
			tfvarsObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: expectedTFVarsName, Labels: labels}
			stateObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: expectedStateName, Labels: labels}

			expectedConfiguration = &corev1.ConfigMap{
				ObjectMeta: configurationObjectMeta,
				Data: map[string]string{
					TerraformConfigMainKey: main,
					TerraformConfigVarsKey: variables,
				},
			}
			expectedInitState = &corev1.ConfigMap{
				ObjectMeta: stateObjectMeta,
				Data: map[string]string{
					TerraformStateKey: "",
				},
			}
			expectedTFVars = &corev1.Secret{
				ObjectMeta: tfvarsObjectMeta,
				Data: map[string][]byte{
					TerraformTFVarsKey: []byte(tfvars),
				},
			}
		})
		Context("Create Config", func() {
			It("Should create the configuration ConfigMap", func() {
				gomock.InOrder(
					fakeClient.EXPECT().
						Get(gomock.Any(), configurationKey, &corev1.ConfigMap{ObjectMeta: configurationObjectMeta}).
						Return(apierrors.NewNotFound(configMapGroupResource, expectedConfigurationName)),
					fakeClient.EXPECT().
						Create(gomock.Any(), expectedConfiguration.DeepCopy()),
				)

				actual, err := tfr.createOrUpdateConfigurationConfigMap(ctx, main, variables, []byte{})
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expectedConfiguration))
			})
			It("Should create the initial state ConfigMap", func() {
				gomock.InOrder(
					fakeClient.EXPECT().
						Get(gomock.Any(), stateKey, &corev1.ConfigMap{ObjectMeta: stateObjectMeta}).
						Return(apierrors.NewNotFound(configMapGroupResource, expectedStateName)),
					fakeClient.EXPECT().
						Create(gomock.Any(), expectedInitState.DeepCopy()),
				)

				actual, err := tfr.createStateConfigMap(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expectedInitState))
			})
			It("Should create the TFVars Secret", func() {
				gomock.InOrder(
					fakeClient.EXPECT().
						Get(gomock.Any(), tfvarsKey, &corev1.Secret{ObjectMeta: tfvarsObjectMeta}).
						Return(apierrors.NewNotFound(secretGroupResource, expectedTFVarsName)),
					fakeClient.EXPECT().
						Create(gomock.Any(), expectedTFVars.DeepCopy()),
				)
				actual, err := tfr.createOrUpdateTFVarsSecret(ctx, tfvars)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expectedTFVars))
			})
		})
		Context("Delete Config", func() {
			It("Should clean up the configuration, tfvars and state resources", func() {
				gomock.InOrder(
					fakeClient.EXPECT().Delete(gomock.Any(), &corev1.Secret{ObjectMeta: tfvarsObjectMeta}),
					fakeClient.EXPECT().Delete(gomock.Any(), &corev1.ConfigMap{ObjectMeta: configurationObjectMeta}),
					fakeClient.EXPECT().Delete(gomock.Any(), &corev1.ConfigMap{ObjectMeta: stateObjectMeta}),
				)
				err := tfr.cleanUpConfig(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Describe("State", func() {
			Context("Get outputs", func() {
				It("Should fail if state key is missing", func() {
					state := map[string]interface{}{
						"version": 4,
					}
					stateJSON, err := json.Marshal(state)
					Expect(err).NotTo(HaveOccurred())

					fakeClient.EXPECT().
						Get(gomock.Any(), stateKey, &corev1.ConfigMap{}).
						DoAndReturn(func(_ context.Context, _ client.ObjectKey, cm *corev1.ConfigMap) error {
							cm.Data = map[string]string{
								TerraformStateKey: string(stateJSON),
							}
							return nil
						})
					actual, err := tfr.GetOutputFromState(ctx)
					Expect(actual).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
				It("Should fail if state is empty", func() {
					fakeClient.EXPECT().
						Get(gomock.Any(), stateKey, &corev1.ConfigMap{}).
						DoAndReturn(func(_ context.Context, _ client.ObjectKey, cm *corev1.ConfigMap) error {
							cm.Data = map[string]string{
								TerraformStateKey: "",
							}
							return nil
						})
					actual, err := tfr.GetOutputFromState(ctx)
					Expect(actual).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
				It("Should fail if config map does not exist", func() {
					fakeClient.EXPECT().
						Get(gomock.Any(), stateKey, &corev1.ConfigMap{}).
						DoAndReturn(func(_ context.Context, _ client.ObjectKey, cm *corev1.ConfigMap) error {
							return apierrors.NewNotFound(configMapGroupResource, tfr.StateConfigMapName)
						})
					actual, err := tfr.GetOutputFromState(ctx)
					Expect(actual).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
				It("Should get the state", func() {
					var (
						state = map[string]interface{}{
							"version": 4,
							"outputs": map[string]interface{}{
								"key1": map[string]interface{}{
									"value": []string{"var1", "var2"},
									"type": []interface{}{
										"tuple",
										[]string{"string"},
									}}}}
						expectedJSON json.RawMessage
						expected     = map[string]interface{}{
							"key1": map[string]interface{}{
								"value": []string{"var1", "var2"},
								"type": []interface{}{
									"tuple",
									[]string{"string"},
								}}}
					)
					stateJSON, err := json.Marshal(state)
					Expect(err).NotTo(HaveOccurred())
					expectedJSON, err = json.Marshal(expected)
					Expect(err).NotTo(HaveOccurred())

					fakeClient.EXPECT().
						Get(gomock.Any(), kutils.ObjectKey(tfr.StateConfigMapName, tfr.Namespace), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
						DoAndReturn(func(_ context.Context, _ client.ObjectKey, cm *corev1.ConfigMap) error {
							cm.Data = map[string]string{
								TerraformStateKey: string(stateJSON),
							}
							return nil
						})
					actual, err := tfr.GetOutputFromState(ctx)
					Expect(actual).To(Equal(expectedJSON))
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})

	Describe("#Pod", func() {
		var (
			serviceAccountKey client.ObjectKey
			roleKey           client.ObjectKey
			roleBindingKey    client.ObjectKey
			podKey            client.ObjectKey

			serviceAccountObjectMeta metav1.ObjectMeta
			roleObjectMeta           metav1.ObjectMeta
			roleBindingObjectMeta    metav1.ObjectMeta
			podObjectMeta            metav1.ObjectMeta

			expectedPod            *corev1.Pod
			expectedServiceAccount *corev1.ServiceAccount
			expectedRole           *rbacv1.Role
			expectedRoleBinding    *rbacv1.RoleBinding

			labels map[string]string
		)

		BeforeEach(func() {
			labels = map[string]string{
				LabelKeyItemName:      itemName,
				LabelKeyItemNamespace: itemNamespace,
			}

			serviceAccountKey = kutils.ObjectKey(expectedServiceAccountName, namespace)
			roleKey = kutils.ObjectKey(expectedRoleName, namespace)
			roleBindingKey = kutils.ObjectKey(expectedRoleBindingName, namespace)
			podKey = kutils.ObjectKey(expectedPodName, namespace)

			serviceAccountObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: expectedServiceAccountName, Labels: labels}
			roleObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: expectedRoleName, Labels: labels}
			roleBindingObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: expectedRoleBindingName, Labels: labels}
			podObjectMeta = metav1.ObjectMeta{
				Namespace: namespace,
				Labels:    labels,
			}

			expectedServiceAccount = &corev1.ServiceAccount{ObjectMeta: serviceAccountObjectMeta}
			expectedRole = &rbacv1.Role{
				ObjectMeta: roleObjectMeta,
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "secrets"},
					Verbs:     []string{"*"},
				}},
			}
			expectedRoleBinding = &rbacv1.RoleBinding{
				ObjectMeta: roleBindingObjectMeta,
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "Role",
					Name:     expectedRoleName,
				},
				Subjects: []rbacv1.Subject{{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      expectedServiceAccountName,
					Namespace: namespace,
				}},
			}

		})
		Describe("Create RBAC", func() {
			It("Should create the ServiceAccount", func() {
				gomock.InOrder(
					fakeClient.EXPECT().
						Get(gomock.Any(), serviceAccountKey, &corev1.ServiceAccount{ObjectMeta: serviceAccountObjectMeta}).
						Return(apierrors.NewNotFound(serviceAccountGroupResource, tfr.Name)),
					fakeClient.EXPECT().
						Create(gomock.Any(), expectedServiceAccount.DeepCopy()),
				)
				err := tfr.createOrUpdateServiceAccount(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create the Role", func() {
				gomock.InOrder(
					fakeClient.EXPECT().
						Get(gomock.Any(), roleKey, &rbacv1.Role{ObjectMeta: roleObjectMeta}).
						Return(apierrors.NewNotFound(roleGroupResource, tfr.Name)),
					fakeClient.EXPECT().
						Create(gomock.Any(), expectedRole.DeepCopy()),
				)
				err := tfr.createOrUpdateRole(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create the RoleBinding", func() {
				gomock.InOrder(
					fakeClient.EXPECT().
						Get(gomock.Any(), roleBindingKey, &rbacv1.RoleBinding{ObjectMeta: roleBindingObjectMeta}).
						Return(apierrors.NewNotFound(roleBindingGroupResource, tfr.Name)),
					fakeClient.EXPECT().
						Create(gomock.Any(), expectedRoleBinding.DeepCopy()),
				)
				err := tfr.createOrUpdateRoleBinding(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Describe("Delete RBAC", func() {
			It("Should delete the ServiceAccount, Role and Rolebinding", func() {
				gomock.InOrder(
					fakeClient.EXPECT().Delete(gomock.Any(), &corev1.ServiceAccount{ObjectMeta: serviceAccountObjectMeta}),
					fakeClient.EXPECT().Delete(gomock.Any(), &rbacv1.Role{ObjectMeta: roleObjectMeta}),
					fakeClient.EXPECT().Delete(gomock.Any(), &rbacv1.RoleBinding{ObjectMeta: roleBindingObjectMeta}),
				)
				err := tfr.cleanUpRBAC(ctx)
				Expect(err).NotTo(HaveOccurred())
			})

			Describe("Manage pods", func() {
				It("Should list the running terraformer pods", func() {
					var (
						podList = &corev1.PodList{}
					)
					gomock.InOrder(
						fakeClient.EXPECT().List(gomock.Any(), podList, client.InNamespace(namespace), client.MatchingLabels(labels)),
					)
					_, err := tfr.listTerraformerPods(ctx)
					Expect(err).NotTo(HaveOccurred())
				})
				It("Should create a new terraformer pod", func() {
					command := "command"

					podObjectMeta.Name = expectedPodName
					podObjectMeta.Labels[LabelKeyGeneration] = strconv.FormatInt(itemGeneration, 10)
					podObjectMeta.Labels[LabelKeyCommand] = command
					expectedPod = &corev1.Pod{
						ObjectMeta: podObjectMeta,
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:            BaseName,
								Image:           terraformContainer.Image,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command: []string{
									"/terraformer",
									command,
									"--zap-log-level=" + logLevel,
									"--configuration-configmap-name=" + expectedConfigurationName,
									"--state-configmap-name=" + expectedStateName,
									"--variables-secret-name=" + expectedTFVarsName,
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("200Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("500m"),
										corev1.ResourceMemory: resource.MustParse("1.5Gi"),
									},
								},
							}},
							RestartPolicy:                 corev1.RestartPolicyNever,
							ServiceAccountName:            expectedServiceAccountName,
							TerminationGracePeriodSeconds: pointer.Int64Ptr(TerminationGracePeriodSeconds),
						}}

					gomock.InOrder(
						fakeClient.EXPECT().Create(gomock.Any(), expectedPod.DeepCopy()),
					)

					actual, err := tfr.createPod(ctx, command, itemGeneration)
					Expect(err).NotTo(HaveOccurred())
					Expect(actual).To(Equal(expectedPod))

				})
				It("Should get the pod an return nil and an error if it does not exist", func() {
					gomock.InOrder(
						fakeClient.EXPECT().
							Get(gomock.Any(), podKey, &corev1.Pod{}).
							Return(apierrors.NewNotFound(podGroupResource, tfr.Name)),
					)
					actual, err := tfr.GetPod(ctx)
					Expect(err).To(HaveOccurred())
					Expect(actual).To(BeNil())
				})
				It("Should get the pod", func() {
					pod := &corev1.Pod{}
					gomock.InOrder(
						fakeClient.EXPECT().
							Get(gomock.Any(), podKey, pod),
					)
					actual, err := tfr.GetPod(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(actual).To(Equal(pod))
				})
				It("Should not fail deleting the terraformer pods if none are running", func() {
					var podList = &corev1.PodList{}
					for _, pod := range podList.Items {
						gomock.InOrder(
							fakeClient.EXPECT().Delete(gomock.Any(), &pod),
						)
					}
					err := tfr.deletePods(ctx, podList)
					Expect(err).NotTo(HaveOccurred())
				})
				It("Should delete the terraformer pod", func() {
					var podList = &corev1.PodList{
						Items: []corev1.Pod{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:      expectedPodName,
									Namespace: namespace,
									Labels:    labels,
								},
							},
						},
					}
					for _, pod := range podList.Items {
						gomock.InOrder(
							fakeClient.EXPECT().Delete(gomock.Any(), &pod),
						)
					}
					err := tfr.deletePods(ctx, podList)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

	})
})
