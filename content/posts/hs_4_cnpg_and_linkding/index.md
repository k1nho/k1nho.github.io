---
title: "Kinho's Homelab Series - PostgreSQL Database and Linkding"
pubDate: 2026-03-20
Description: "Let's build a mini homelab! In this entry, we explore the cloud native postgres operator for managing postgres databases and run the Linkding application!"
Categories: ["DevOps", "Networking", "Platform Engineering"]
Tags: ["Homelab Series", "DevOps", "GitOps", "Databases"]
cover: "gallery/homelabs_cover4.png"
images:
  - "gallery/homelab_in4.png"
mermaid: true
draft: true
---

Welcome to another entry in **Kinho's Homelab Series**! In the last [entry]({{<relref "hs_3_argocd_secrets_firstapp/index.md">}}), we created the foundation for all our Kubernetes
infrastructure by adopting the GitOps workflow with Argo CD. So far, I have deployed the blog which belongs in the category of [stateless applications]().
It is time to kick it up a notch by running [stateful applications]()

In this entry, I'll go over running **PostgreSQL in Kubernetes**, and setup **the Linkding bookmark manager** application to use it. Along the way, I discuss why you want
to avoid the usual _recommended_ way of running stateful applications in the form of the **StatefulSet**, and opt for a more reliable and proven management, in this case of
PostgreSQL, with the use of an **operator**.

---

# 3. PostgreSQL databases and Linkding

![Homelab Series 4](gallery/homelab_in4.png)

---

## Kubernetes and Databases: a Debate for the Ages

Just like editor, or language wars, storage has always been a topic of debate in Kubernetes. By default, the Kubernetes documentation explains that using the **StatefulSet** object is useful for managing applications
that need persistent storage:

> A StatefulSet runs a group of Pods, and maintains a sticky identity for each of those Pods. This is useful for managing applications that need persistent storage or a stable, unique network identity.

However, Kubernetes has always been know as a **container orchestrator**, although I like to think about it as [data types for your infrastructure](), and by the nature
of containers therefore ephemeral, meaning that it contradicts the very own nature of storage, that is, being persistent. The challenge has sparked the debate since the birth of Kuberentes of whether or
not databases should even be ran on Kubernetes in the first place. This conumdrum has resulted in people usually looking for alternatives such as migrating the database service to a dedicated server outside of the k8s cluster, or
opt-out for a cloud manage service for an specific database.

The conclusion should simply be to avoid databases in Kubernetes at all costs, right? Recently, while watching the talk [Why are we still talking about containers?](https://www.youtube.com/watch?v=x1t2GPChhX8) by the distinguish Google engineer Kelsey Hightower,
a question comes up:

> Should we run databases in Kubernetes?

Kelsey's answer goes something like this:

> It used to be no, because I watched people lose their data. My guess is people are still losing their data! but the tools are just so much better: **volume snapshotting, restoring, and replication**.

Yes, containers are ephemeral, yes, data can be easily loss if you run StatefulSets blindly. Then, is it even possible to reach a solution that would enable us easily have **the tooling** in place
such that it combines the knowledge of Kubernetes patterns, and deep expertise of database management. Fortunately, with Kubernetes adoption across multiple cloud providers and on-premise, many
have set out the task to do just that by using the powerful: [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

---

## The Operator Pattern

One of the most powerful, if not the best, pattern in Kubernetes is the operator pattern, but what is it? From Kubernetes official docs:

> The operator pattern aims to capture the key aim of a human operator who is managing a service or set of services. Human operators who look after specific applications and services have deep knowledge of how the system ought to behave, how to deploy it, and how to react if there are problems.

Wow isn't that just what we would like to have for our databases! human operators with deep knowledge of how a system out to behave, except there is no human and it is automated
to handle the scenarios where things eventually can go wrong. To that end, the operator pattern has emerged as the best way to manage databases in Kubernetes.

---

## PostgreSQL Databases in Kubernetes (CloudNativePG Operator)

![CloudNativePG Logo](gallery/CNPG.png)

Today, I'll take a look at the [CloudNativePG](https://cloudnative-pg.io/) project for my homelab to start running PostgreSQL databases. The **CloudNativePG** operator
is an extremely powerful operator that encompasses the full operational lifecycle of a highly available PostgreSQL cluster. Initially, it was built by experts over at [EnterpriseDB](https://www.enterprisedb.com/products/edb-postgres-ai-for-cloudnativepg),
and eventually became an [incubated project](https://landscape.cncf.io/?item=app-definition-and-development--database--cloudnativepg) under the CNCF.

---

---

---

## Wrapping up

- **Previous:** [Kinho's Homelab Series - Orchestration Platform and Networking (K3s + Cilium)]({{< relref "hs_2_k3scilium/index.md">}})
- **Next:** TBD

---

## Resources
