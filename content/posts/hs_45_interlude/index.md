---
title: "Kinho's Homelab Series - Interlude I (Argo Promotion System, and Debug Log)"
pubDate: 2026-05-20
description: "Let's build a mini homelab! In this entry, we explore the cloud-native PostgreSQL operator for managing PostgreSQL clusters and run the Linkding application!"
Categories: ["DevOps", "Networking", "Platform Engineering", "Databases"]
Tags: ["Homelab Series", "DevOps", "GitOps", "Databases"]
cover: "gallery/homelabs_cover4.png"
images:
  - "gallery/homelab_in4.png"
mermaid: true
draft: true
---

# Argo Structure: From Flat to Promotion System

I moved from the flat implementation of the app of apps pattern (app of apps -> argocd application) into
the promotion/environment system structure a.k.a the 3-tier structure (app of apps -> appset -> kubernetes manifest/ helm configs).
This structure enables more flexible setups that provide better management for multi-cluster setups. The key addition are [argo appsets](https://argo-cd.readthedocs.io/en/latest/user-guide/application-set/) which in turn automatically create and deliver the corresponding applications to different clusters labeled accordingly to a hierarchy.
These three articles by CodeFresh were a fantastic read:

- [How to Structure Your Argo CD Repositories Using Application Sets](https://codefresh.io/blog/how-to-structure-your-argo-cd-repositories-using-application-sets/)
- [Using Helm Hierarchies in Multi-Source Argo CD Applications for Promoting to Different GitOps Environments](https://codefresh.io/blog/helm-values-argocd/)
- [Distribute Your Argo CD Applications to Different Kubernetes Clusters Using Application Sets](https://codefresh.io/blog/argocd-clusters-labels-with-apps/)

This structure will now enable me to replicate apps in my homelab cluster to other environments including cloud. The idea here is
that environments can be translated by a simple directory copy. I can then control the values of the manifest such as resources, replicas, etc or the helm chart values
that suits the needs of the environment they will be ran on.

# Debug Log

## The TLSRoute breakage

After upgrading the kernel and taking care of [CVE-2026-31431](https://copy.fail/), ran into an interesting breakage of
networking, the cilium operator was stuck and therefore most of the core components including coredns, local-path, and metrics-server
where down. The issue stem from not having the correct installation of the Gateway API crds when having `gatewayAPI.enable=true` in the
cilium chart installation.

```terminal
level=error msg="Invoke failed" error="failed to create gateway controller: failed to setup reconciler:
failed to setup field indexer \"backendServiceTLSRouteIndex\":
no matches for kind \"TLSRoute\" in version \"gateway.networking.k8s.io/v1alpha2\""
function="gateway-api.initGatewayAPIController (pkg/gateway-api/cell.go:108)"
```

## The MTU misconfig

The services on tailscale were incredibly slow this subtle misconfiguration caused all of it

```bash
kubectl -n kube-system exec ds/cilium -- cilium status --verbose | grep -i mtu
```

This would show the MTU as something like this

```terminal
├── mtu
│   ├── job-endpoint-mtu-updater                        [OK] Endpoint MTU updated (20s, x1)
│   └── job-mtu-updater                                 [OK] MTU updated (1280) (21s, x1)
```

The eBPF datapath was going through the tailscale interface which notably had a max transmission unit (MTU) of 1280.
This resulted in the services not loading or loading very slowly due to the packet transmission being not the most optimal.
Luckily, the fix is quite easy, just explicitly list the nic that the eBPF datapath should go through.
