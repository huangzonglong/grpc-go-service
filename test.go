package main

import (
	"fmt"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	http_router "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	http_conn_manager "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/envoy/type"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	proto_type "github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"net"
	"time"
)

type NodeConfig struct {
	node      *core.Node
	endpoints []cache.Resource //[]*api.ClusterLoadAssignment
	clusters  []cache.Resource //[]*api.Cluster
	routes    []cache.Resource //[]*api.RouteConfiguration
	listeners []cache.Resource //[]*api.Listener
}

//implement cache.NodeHash
func (n NodeConfig) ID(node *core.Node) string {
	return node.GetId()
}

func ADD_Cluster_With_Static_Endpoint(n *NodeConfig) {
	cluster := &api.Cluster{
		Name:           "cluster_with_static_endpoint",
		ConnectTimeout: 1 * time.Second,
		LbPolicy:       api.Cluster_ROUND_ROBIN,
		LoadAssignment:&api.ClusterLoadAssignment{
			ClusterName: "none",
			Endpoints: []endpoint.LocalityLbEndpoints{
				endpoint.LocalityLbEndpoints{
					LbEndpoints: []endpoint.LbEndpoint{
						endpoint.LbEndpoint{
							//Endpoint: &endpoint.Endpoint{
							//	Address: &core.Address{
							//		Address: &core.Address_SocketAddress{
							//			SocketAddress: &core.SocketAddress{
							//				Protocol: core.TCP,
							//				Address:  "0.0.0.0",
							//				PortSpecifier: &core.SocketAddress_PortValue{
							//					PortValue: 80,
							//				},
							//			},
							//		},
							//	},
							//},
						},
					},
				},
			},
		},
	}
	n.clusters = append(n.clusters, cluster)
}


func ADD_Cluster_With_Dynamic_Endpoint(n *NodeConfig) {
	endpoint := &api.ClusterLoadAssignment{
		ClusterName: "cluster_with_dynamic_endpoint",
		Endpoints: []endpoint.LocalityLbEndpoints{
			endpoint.LocalityLbEndpoints{
				LbEndpoints: []endpoint.LbEndpoint{
					endpoint.LbEndpoint{
						//Endpoint: &endpoint.Endpoint{
						//	Address: &core.Address{
						//		Address: &core.Address_SocketAddress{
						//			SocketAddress: &core.SocketAddress{
						//				Protocol: core.TCP,
						//				Address:  "0.0.0.0",
						//				PortSpecifier: &core.SocketAddress_PortValue{
						//					PortValue: 80,
						//				},
						//			},
						//		},
						//	},
						//},
					},
				},
			},
		},
		Policy: &api.ClusterLoadAssignment_Policy{
			DropOverloads: []*api.ClusterLoadAssignment_Policy_DropOverload{
				&api.ClusterLoadAssignment_Policy_DropOverload{
					Category: "drop_policy1",
					DropPercentage: &envoy_type.FractionalPercent{
						Numerator:   3,
						Denominator: envoy_type.FractionalPercent_HUNDRED,
					},
				},
			},
			OverprovisioningFactor: &proto_type.UInt32Value{
				Value: 140,
			},
		},
	}

	cluster := &api.Cluster{
		Name:           "cluster_with_dynamic_endpoint",
		ConnectTimeout: 1 * time.Second,
		LbPolicy:       api.Cluster_ROUND_ROBIN,
		EdsClusterConfig: &api.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
					ApiConfigSource: &core.ApiConfigSource{
						ApiType: core.ApiConfigSource_GRPC,
						GrpcServices: []*core.GrpcService{
							&core.GrpcService{
								TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
										ClusterName: "xds_cluster",
									},
								},
							},
						},
					},
				},
			},
			//            ServiceName: "dynamic_endpoints", //与endpoint中的ClusterName对应。
		},
	}

	n.endpoints = append(n.endpoints, endpoint)
	n.clusters = append(n.clusters, cluster)
}


func ADD_Listener_With_Static_Route(n *NodeConfig) {
	http_filter_router_ := &http_router.Router{
		DynamicStats: &proto_type.BoolValue{
			Value: true,
		},
	}
	http_filter_router, err := util.MessageToStruct(http_filter_router_)
	if err != nil {
		glog.Error(err)
		return
	}

	listen_filter_http_conn_ := &http_conn_manager.HttpConnectionManager{
		StatPrefix: "ingress_http",
		RouteSpecifier: &http_conn_manager.HttpConnectionManager_RouteConfig{
			RouteConfig: &api.RouteConfiguration{
				Name: "None",
				VirtualHosts: []route.VirtualHost{
					route.VirtualHost{
						Name: "local",
						Domains: []string{
							"webshell.com",
						},
						Routes: []route.Route{
							route.Route{
								Match: route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Prefix{
										Prefix: "/",
									},
									CaseSensitive: &proto_type.BoolValue{
										Value: false,
									},
								},
								Action: &route.Route_Route{
									Route: &route.RouteAction{
										ClusterSpecifier: &route.RouteAction_Cluster{
											Cluster: "cluster_with_static_endpoint",
										},
										HostRewriteSpecifier: &route.RouteAction_HostRewrite{
											HostRewrite: "webshell.com",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		HttpFilters: []*http_conn_manager.HttpFilter{
			&http_conn_manager.HttpFilter{
				Name: "envoy.router",
				ConfigType: &http_conn_manager.HttpFilter_Config{
					Config: http_filter_router,
				},
			},
		},
	}
	listen_filter_http_conn, err := util.MessageToStruct(listen_filter_http_conn_)
	if err != nil {
		glog.Error(err)
		return
	}

	listener := &api.Listener{
		Name: "listener_with_static_route_port_9000",
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 9000,
					},
				},
			},
		},
		FilterChains: []listener.FilterChain{
			listener.FilterChain{
				Filters: []listener.Filter{
					listener.Filter{
						Name: "envoy.http_connection_manager",
						ConfigType: &listener.Filter_Config{
							Config: listen_filter_http_conn,
						},
					},
				},
			},
		},
	}

	n.listeners = append(n.listeners, listener)
}


func ADD_Listener_With_Dynamic_Route(n *NodeConfig) {
	route := &api.RouteConfiguration{
		Name: "dynamic_route",
		VirtualHosts: []route.VirtualHost{
			route.VirtualHost{
				Name: "local",
				Domains: []string{
					"webshell.com",
				},
				Routes: []route.Route{
					route.Route{
						Match: route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
							CaseSensitive: &proto_type.BoolValue{
								Value: false,
							},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_Cluster{
									Cluster: "cluster_with_static_endpoint",
								},
								HostRewriteSpecifier: &route.RouteAction_HostRewrite{
									HostRewrite: "webshell.com",
								},
							},
						},
					},
				},
			},
		},
	}

	http_filter_router_ := &http_router.Router{
		DynamicStats: &proto_type.BoolValue{
			Value: true,
		},
	}
	http_filter_router, err := util.MessageToStruct(http_filter_router_)
	if err != nil {
		glog.Error(err)
		return
	}

	listen_filter_http_conn_ := &http_conn_manager.HttpConnectionManager{
		StatPrefix: "ingress_http",
		RouteSpecifier: &http_conn_manager.HttpConnectionManager_Rds{
			Rds: &http_conn_manager.Rds{
				RouteConfigName: "dynamic_route",
				ConfigSource: core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
						ApiConfigSource: &core.ApiConfigSource{
							ApiType: core.ApiConfigSource_GRPC,
							GrpcServices: []*core.GrpcService{
								&core.GrpcService{
									TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
										EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
											ClusterName: "xds_cluster",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		HttpFilters: []*http_conn_manager.HttpFilter{
			&http_conn_manager.HttpFilter{
				Name: "envoy.router",
				ConfigType: &http_conn_manager.HttpFilter_Config{
					Config: http_filter_router,
				},
			},
		},
	}
	listen_filter_http_conn, err := util.MessageToStruct(listen_filter_http_conn_)
	if err != nil {
		glog.Error(err)
		return
	}

	listener := &api.Listener{
		Name: "listener_with_dynamic_route_port_9001",
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 9001,
					},
				},
			},
		},
		FilterChains: []listener.FilterChain{
			listener.FilterChain{
				Filters: []listener.Filter{
					listener.Filter{
						Name: "envoy.http_connection_manager",
						ConfigType: &listener.Filter_Config{
							Config: listen_filter_http_conn,
						},
					},
				},
			},
		},
	}

	n.listeners = append(n.listeners, listener)
	n.routes = append(n.routes, route)
}

func Update_SnapshotCache(s cache.SnapshotCache, n *NodeConfig, version string) {
	err := s.SetSnapshot(n.ID(n.node), cache.NewSnapshot(version, n.endpoints, n.clusters, n.routes, n.listeners))
	if err != nil {
		glog.Error(err)
	}
}

func main() {
	c := &api.Cluster{

	}

	snapshotCache := cache.NewSnapshotCache(false, NodeConfig{}, nil)
	server := xds.NewServer(snapshotCache, nil)//
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":5678")

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			// error handling
		}
	}()

	node := &core.Node{
		Id:      "envoy-64.58",
		Cluster: "test",
	}

	node_config := &NodeConfig{
		node:      node,
		endpoints: []cache.Resource{}, //[]*api.ClusterLoadAssignment
		clusters:  []cache.Resource{}, //[]*api.Cluster
		routes:    []cache.Resource{}, //[]*api.RouteConfiguration
		listeners: []cache.Resource{}, //[]*api.Listener
	}

	input := ""
	fmt.Println()

	fmt.Printf("Enter to update version 1: ADD_Cluster_With_Static_Endpoint")
	fmt.Scanf("\n", &input)
	ADD_Cluster_With_Static_Endpoint(node_config)
	Update_SnapshotCache(snapshotCache, node_config, "1")
	fmt.Printf("ok")


	fmt.Printf("\nEnter to update version 2: ADD_Cluster_With_Dynamic_Endpoint")
	fmt.Scanf("\n", &input)
	ADD_Cluster_With_Dynamic_Endpoint(node_config)
	Update_SnapshotCache(snapshotCache, node_config, "2")
	fmt.Printf("ok")

	fmt.Printf("\nEnter to update version 3: ADD_Listener_With_Static_Route")
	fmt.Scanf("\n", &input)
	ADD_Listener_With_Static_Route(node_config)
	Update_SnapshotCache(snapshotCache, node_config, "3")
	fmt.Printf("ok")

	fmt.Printf("\nEnter to update version 4: ADD_Listener_With_Dynamic_Route")
	fmt.Scanf("\n", &input)
	ADD_Listener_With_Dynamic_Route(node_config)
	Update_SnapshotCache(snapshotCache, node_config, "4")
	fmt.Printf("ok")

	fmt.Printf("\nEnter to exit: ")
	fmt.Scanf("\n", &input)
}