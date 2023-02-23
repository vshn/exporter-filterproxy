package target

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesEndpointFetcher struct {
	endpointname string
	namespace    string

	port   int
	path   string
	scheme string

	client    *http.Client
	authToken string

	kube client.Client

	clock           func() time.Time
	refreshInterval time.Duration
	mutex           sync.Mutex
	cache           map[string][]dto.MetricFamily
	lastUpdated     time.Time
}

type KubernetesEndpointFetcherOpts struct {
	Endpointname string
	Namespace    string

	Port   int
	Path   string
	Scheme string

	AuthToken          string
	RefreshInterval    time.Duration
	InsecureSkipVerify bool
}

func NewKubernetesEndpointFetcher(opts KubernetesEndpointFetcherOpts) (*KubernetesEndpointFetcher, error) {
	restConf, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := client.New(restConf, client.Options{})
	if err != nil {
		return nil, err
	}

	return &KubernetesEndpointFetcher{
		endpointname: opts.Endpointname,
		namespace:    opts.Namespace,
		port:         opts.Port,
		path:         opts.Path,
		scheme:       opts.Scheme,

		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: opts.InsecureSkipVerify,
				},
			},
		},
		authToken: opts.AuthToken,

		kube: kubeClient,

		refreshInterval: opts.RefreshInterval,
		mutex:           sync.Mutex{},
		cache:           map[string][]dto.MetricFamily{},
		lastUpdated:     time.Time{},
	}, nil
}

func (f *KubernetesEndpointFetcher) FetchMetricsFor(ctx context.Context, endpoint string) ([]dto.MetricFamily, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.now().Sub(f.lastUpdated) < f.refreshInterval {
		return f.cache[endpoint], nil
	}

	endpoints, err := f.discover(ctx)
	if err != nil {
		return nil, err
	}

	g, _ := errgroup.WithContext(ctx)
	m := sync.Mutex{}
	for _, ip := range endpoints {
		ip := ip
		g.Go(func() error {
			metrics, err := fetchMetrics(f.client, f.buildAddr(ip), f.authToken)
			if err != nil {
				return err
			}
			m.Lock()
			f.cache[ip] = metrics
			m.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	f.lastUpdated = f.now()
	return f.cache[endpoint], nil
}

func (f *KubernetesEndpointFetcher) now() time.Time {
	if f.clock != nil {
		return f.clock()
	}
	return time.Now()
}

func (f *KubernetesEndpointFetcher) buildAddr(ip string) string {
	return fmt.Sprintf("%s://%s:%d%s", f.scheme, ip, f.port, f.path)
}

func (f *KubernetesEndpointFetcher) discover(ctx context.Context) ([]string, error) {

	ep := corev1.Endpoints{}
	err := f.kube.Get(ctx, types.NamespacedName{
		Namespace: f.namespace,
		Name:      f.endpointname,
	}, &ep)
	if err != nil {
		return nil, err
	}

	epIPs := []string{}
	seenIps := map[string]bool{}
	for _, subset := range ep.Subsets {
		if !hasPort(subset, f.port) {
			continue
		}
		for _, addr := range subset.Addresses {
			if seenIps[addr.IP] {
				continue
			}
			epIPs = append(epIPs, addr.IP)
			seenIps[addr.IP] = true
		}
	}

	return epIPs, nil
}

func hasPort(subset corev1.EndpointSubset, p int) bool {
	for _, port := range subset.Ports {
		if int(port.Port) == p {
			return true
		}
	}
	return false
}
