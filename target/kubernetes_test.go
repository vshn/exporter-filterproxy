package target

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKube_Fetch(t *testing.T) {
	sa := startTestTarget(t, "../testdata/simple", "127.0.8.1:8911")
	defer sa.Close()
	sb := startTestTarget(t, "../testdata/simpletwo", "127.0.8.2:8911")
	defer sb.Close()

	f := KubernetesEndpointFetcher{
		endpointname: "test-ep",
		namespace:    "fetch-test",
		port:         8911,
		path:         "/",
		scheme:       "http",
		client:       sa.Client(),
		kube: newTestKubeEnv(
			newTestEndpoint(8911, "127.0.8.1", "127.0.8.2"),
		),
		cache: map[string][]dto.MetricFamily{},
	}

	metrics, err := f.FetchMetricsFor(context.TODO(), "127.0.8.1")
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	// The decoder provided by prometheus doesn't return metric families in a deterministic order
	// so we need to compare them in a order independent way
	mfs := map[string]*dto.MetricFamily{}
	for i, mf := range metrics {
		mfs[mf.GetName()] = &metrics[i]
	}
	assert.Contains(t, mfs, "test_metric_one")
	assert.Len(t, mfs["test_metric_one"].GetMetric(), 3)
	assert.Equal(t, 0.3, mfs["test_metric_one"].GetMetric()[2].Gauge.GetValue())

	assert.Contains(t, mfs, "test_metric_two")
	assert.Len(t, mfs["test_metric_two"].GetMetric(), 3)
	assert.EqualValues(t, 3, mfs["test_metric_two"].GetMetric()[2].Counter.GetValue())

	metrics, err = f.FetchMetricsFor(context.TODO(), "127.0.8.2")
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	// The decoder provided by prometheus doesn't return metric families in a deterministic order
	// so we need to compare them in a order independent way
	mfs = map[string]*dto.MetricFamily{}
	for i, mf := range metrics {
		mfs[mf.GetName()] = &metrics[i]
	}
	assert.Contains(t, mfs, "test_metric_one")
	assert.Len(t, mfs["test_metric_one"].GetMetric(), 3)
	assert.Equal(t, 4.3, mfs["test_metric_one"].GetMetric()[2].Gauge.GetValue())

	assert.Contains(t, mfs, "test_metric_two")
	assert.Len(t, mfs["test_metric_two"].GetMetric(), 3)
	assert.EqualValues(t, 31, mfs["test_metric_two"].GetMetric()[2].Counter.GetValue())
}

func TestKube_FetchCache(t *testing.T) {
	counterA := 0
	counterB := 0
	sa := startTestTarget(t, "../testdata/simple", "127.0.8.1:8911", func() {
		counterA++
	})
	defer sa.Close()
	sb := startTestTarget(t, "../testdata/simpletwo", "127.0.8.2:8911", func() {
		counterB++
	})
	defer sb.Close()

	fakeNow := &time.Time{}
	*fakeNow = time.Now()
	fakeClock := func() time.Time {
		return *fakeNow
	}
	f := KubernetesEndpointFetcher{
		endpointname: "test-ep",
		namespace:    "fetch-test",
		port:         8911,
		path:         "/",
		scheme:       "http",
		client:       sa.Client(),
		kube: newTestKubeEnv(
			newTestEndpoint(8911, "127.0.8.1", "127.0.8.2"),
		),
		cache:           map[string][]dto.MetricFamily{},
		refreshInterval: 5 * time.Second,
		clock:           fakeClock,
	}

	metrics, err := f.FetchMetricsFor(context.TODO(), "127.0.8.1")
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, counterA)
	assert.Equal(t, 1, counterB)
	metrics, err = f.FetchMetricsFor(context.TODO(), "127.0.8.2")
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, counterA)
	assert.Equal(t, 1, counterB)
	metrics, err = f.FetchMetricsFor(context.TODO(), "127.0.8.1")
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, counterA)
	assert.Equal(t, 1, counterB)

	*fakeNow = fakeNow.Add(8 * time.Second)
	metrics, err = f.FetchMetricsFor(context.TODO(), "127.0.8.2")
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 2, counterA)
	assert.Equal(t, 2, counterB)
}

func TestKube_Discover(t *testing.T) {

	tcs := map[string]struct {
		name      string
		namespace string
		port      int

		endpoints []client.Object

		errCheck func(err error) bool
		expected []string
	}{
		"notFound": {
			name:      "test-ep",
			namespace: "fetch-test",
			port:      9001,

			errCheck: apierrors.IsNotFound,
		},
		"simple": {
			name:      "test-ep",
			namespace: "fetch-test",
			port:      9001,
			expected:  []string{"127.0.9.1", "127.0.9.2", "127.0.9.3"},
			endpoints: []client.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ep",
						Namespace: "fetch-test",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP: "127.0.9.1",
								},
								{
									IP: "127.0.9.2",
								},
								{
									IP: "127.0.9.3",
								},
							},
							Ports: []corev1.EndpointPort{
								{
									Name: "metrics",
									Port: int32(9001),
								},
								{
									Name: "metrics",
									Port: int32(9002),
								},
							},
						},
					},
				},
			},
		},
		"noMatchingPort": {
			name:      "test-ep",
			namespace: "fetch-test",
			port:      9008,
			expected:  []string{},
			endpoints: []client.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ep",
						Namespace: "fetch-test",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP: "127.0.9.1",
								},
								{
									IP: "127.0.9.2",
								},
								{
									IP: "127.0.9.3",
								},
							},
							Ports: []corev1.EndpointPort{
								{
									Name: "metrics",
									Port: int32(9001),
								},
								{
									Name: "metrics",
									Port: int32(9002),
								},
							},
						},
					},
				},
			},
		},
		"partialMatch": {
			name:      "test-ep",
			namespace: "fetch-test",
			port:      9002,
			expected:  []string{"127.0.9.1", "127.0.9.2", "127.0.9.5"},
			endpoints: []client.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ep",
						Namespace: "fetch-test",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP: "127.0.9.1",
								},
								{
									IP: "127.0.9.2",
								},
							},
							Ports: []corev1.EndpointPort{
								{
									Name: "metrics",
									Port: int32(9001),
								},
								{
									Name: "metrics",
									Port: int32(9002),
								},
							},
						},
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP: "127.0.9.1",
								},
								{
									IP: "127.0.9.3",
								},
							},
							Ports: []corev1.EndpointPort{
								{
									Name: "metrics",
									Port: int32(9005),
								},
								{
									Name: "metrics",
									Port: int32(9001),
								},
							},
						},
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP: "127.0.9.2",
								},
								{
									IP: "127.0.9.5",
								},
							},
							Ports: []corev1.EndpointPort{
								{
									Name: "metrics",
									Port: int32(9005),
								},
								{
									Name: "metrics",
									Port: int32(9002),
								},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			f := KubernetesEndpointFetcher{
				endpointname: tc.name,
				namespace:    tc.namespace,
				port:         tc.port,
				kube:         newTestKubeEnv(tc.endpoints...),
			}

			targets, err := f.discover(context.TODO())

			if tc.errCheck == nil {
				require.NoError(t, err)
			} else {
				require.True(t, tc.errCheck(err))
			}

			for _, exp := range tc.expected {
				assert.Contains(t, targets, exp)
			}
			assert.Len(t, targets, len(tc.expected))
		})
	}
}

func startTestTarget(t *testing.T, sourceFile string, listenOn string, callback ...func()) *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		data, err := os.ReadFile(sourceFile)
		require.NoError(t, err)
		_, err = rw.Write(data)
		require.NoError(t, err)
		for _, f := range callback {
			f()
		}
	}))
	l, err := net.Listen("tcp", listenOn)
	require.NoError(t, err)
	server.Listener.Close()
	server.Listener = l

	server.Start()
	return server
}

func newTestKubeEnv(obj ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(obj...).
		Build()
}

func newTestEndpoint(port int, ips ...string) *corev1.Endpoints {
	addrs := []corev1.EndpointAddress{}
	for i := range ips {
		addrs = append(addrs, corev1.EndpointAddress{
			IP: ips[i],
		})
	}
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ep",
			Namespace: "fetch-test",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: addrs,
				Ports: []corev1.EndpointPort{
					{
						Name: "metrics",
						Port: int32(port),
					},
				},
			},
		},
	}
}
