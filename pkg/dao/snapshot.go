package dao

import (
	"fmt"
	accessLogConfig "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	accessLoggers "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoyCache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"time"
)

type Status string

const (
	EnvoyHttpMethodHeaderKey string = ":method"

	XdsClusterName string = "gk-grpc"

	StatusActive   = "active"
	StatusInActive = "inactive"
)

type EnvoySnapshot interface {
	// Router
	UnshiftRouter(*ProxyRoute) error
	UpdateRouterByID(id int, newEz *ProxyRoute) error
	ListRouter() ([]ProxyRoute, error)
	RemoveRouterByID(int) error
	// DirectResponse
	UnshiftDirectRes(*DirectResponse) error
	UpdateDirectResByID(id int, newEz *DirectResponse) error
	ListDirectRes() ([]DirectResponse, error)
	RemoveDirectResByID(int) error
	// comm
	GenerateSnapshot() (envoyCache.Snapshot, error)
}

// Proxy
type ProxyRoute struct {
	ID           int      `json:"id"`
	Status       Status   `json:"status"`
	Desc         string   `json:"desc"`
	CreateBy     string   `json:"createBy"`
	Prefix       string   `json:"prefix"`
	ReqMethod    string   `json:"reqMethod"`
	AllowDomains []string `json:"allowDomains"`
	UpstreamName string   `json:"upstreamName"`
	UpstreamHost string   `json:"upstreamHost"`
	UpstreamPort uint32   `json:"upstreamPort"`
}

// Mocker
type DirectResponse struct {
	ID        int    `json:"id"`
	Status    Status `json:"status"`
	Desc      string `json:"desc"`
	CreateBy  string `json:"createBy"`
	Prefix    string `json:"prefix"`
	ReqMethod string `json:"reqMethod"`
	ResStatus int    `json:"resStatus"`
	ResBody   string `json:"resBody"`
}

func parseClustersToResource(clusters []*cluster.Cluster) (r []types.Resource) {
	for _, c := range clusters {
		r = append(r, c)
	}
	return r
}

func makeClusters(easyRoutes []ProxyRoute) []*cluster.Cluster {
	var clusters []*cluster.Cluster

	for _, ez := range easyRoutes {
		clusters = append(clusters, &cluster.Cluster{
			Name:                 ez.UpstreamName,
			LoadAssignment:       makeEndpoint(ez),
			ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_LOGICAL_DNS},
			LbPolicy:             cluster.Cluster_ROUND_ROBIN,
			DnsLookupFamily:      cluster.Cluster_V4_ONLY,
		})
	}

	return clusters
}

func makeEndpoint(easyRoute ProxyRoute) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: easyRoute.UpstreamName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  easyRoute.UpstreamHost,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: easyRoute.UpstreamPort,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

func makeHeaderValueOptions(m map[string]string) (headers []*core.HeaderValueOption) {
	for k, v := range m {
		headers = append(headers, &core.HeaderValueOption{
			Header: &core.HeaderValue{
				Key:   k,
				Value: v,
			},
		})
	}
	return headers
}

func makeRoute(routeName string, easyRoutes []ProxyRoute, directRes []DirectResponse) *route.RouteConfiguration {

	var routes []*route.Route

	for _, dr := range directRes {
		if dr.Status != StatusActive {
			continue
		}
		routes = append(routes, &route.Route{
			ResponseHeadersToAdd: makeHeaderValueOptions(map[string]string{
				"content-type":                "application/json",
				"Access-Control-Allow-Origin": "*",
				"x-mockingbird-type":          "mocker",
				"x-mockingbird-name":          fmt.Sprintf("%s/%d", dr.CreateBy, dr.ID),
			}),
			Match: &route.RouteMatch{
				Headers: []*route.HeaderMatcher{
					{
						Name:                 EnvoyHttpMethodHeaderKey,
						HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{ExactMatch: dr.ReqMethod},
						InvertMatch:          false,
					},
				},
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: dr.Prefix,
				},
			},
			Action: &route.Route_DirectResponse{
				DirectResponse: &route.DirectResponseAction{
					Status: uint32(dr.ResStatus),
					Body: &core.DataSource{
						Specifier: &core.DataSource_InlineString{
							InlineString: dr.ResBody,
						},
					},
				},
			},
		})
	}

	for _, ez := range easyRoutes {
		if ez.Status != StatusActive {
			continue
		}

		var headerMatchers []*route.HeaderMatcher

		if ez.ReqMethod != "*" {
			headerMatchers = append(headerMatchers, &route.HeaderMatcher{
				Name:                 EnvoyHttpMethodHeaderKey,
				HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{ExactMatch: ez.ReqMethod},
				InvertMatch:          false,
			})
		}

		newRoute := &route.Route{
			ResponseHeadersToAdd: makeHeaderValueOptions(map[string]string{
				"x-mockingbird-type": "proxy",
				"x-mockingbird-name": fmt.Sprintf("%s/%d", ez.CreateBy, ez.ID),
			}),
			Match: &route.RouteMatch{
				Headers: headerMatchers,
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: ez.Prefix,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: ez.UpstreamName,
					},
					HostRewriteSpecifier: &route.RouteAction_HostRewriteLiteral{
						HostRewriteLiteral: ez.UpstreamHost,
					},
				},
			},
		}

		routes = append(routes, newRoute)
	}

	return &route.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*route.VirtualHost{
			{
				Name:    fmt.Sprintf("%s_virtual_hosts", routeName),
				Domains: []string{"*"},
				Routes:  routes,
			},
		},
	}
}

func genXdsGrpcServer() *core.GrpcService {
	return &core.GrpcService{
		TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
			EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
				ClusterName: XdsClusterName,
			},
		},
	}
}

func genAccessLogConfig() *accessLoggers.HttpGrpcAccessLogConfig {
	return &accessLoggers.HttpGrpcAccessLogConfig{
		CommonConfig: &accessLoggers.CommonGrpcAccessLogConfig{
			LogName:     "access_log",
			GrpcService: genXdsGrpcServer(),
		},
		AdditionalRequestHeadersToLog: []string{
			"user-agent",
		},
	}
}

func makeHTTPListener(listenerName string, routeName string) *listener.Listener {
	alsConfig, err := ptypes.MarshalAny(genAccessLogConfig())
	if err != nil {
		panic(err)
	}

	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				RouteConfigName: routeName,
				ConfigSource:    makeConfigSource(),
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		AccessLog: []*accessLogConfig.AccessLog{{
			Name: wellknown.HTTPGRPCAccessLog,
			ConfigType: &accessLogConfig.AccessLog_TypedConfig{
				TypedConfig: alsConfig,
			},
		}},
	}
	httpConnManager, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: ListenerPort,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: httpConnManager,
				},
			}},
		}},
	}
}

func makeConfigSource() *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = core.ApiVersion_V3
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       core.ApiVersion_V3,
			ApiType:                   core.ApiConfigSource_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{
				{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "gk-grpc"},
					},
				},
			},
		},
	}
	return source
}
