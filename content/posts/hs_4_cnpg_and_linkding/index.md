---
title: "Kinho's Homelab Series - PostgreSQL Database and Linkding"
pubDate: 2026-05-20
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
have set out the task to do just that by using the powerful: [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

---

### The Operator Pattern

One of the most powerful, if not the most critical, pattern in Kubernetes is the operator pattern, but what is it? From Kubernetes official docs:

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

### Installing the CNPG Operator

As usual, we'll create our Argo `Application` custom resource to install the CNPG operator via GitOps.

> [!NOTE]
> If you missed the previous entry were we go over ArgoCD you can check it [here](({{< relref "hs_2_k3scilium/index.md">}}))

```yaml {filename="argo/cnpg.yaml"}
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"
  name: cnpg-homelab
  namespace: argocd
spec:
  project: default
  sources:
    # Kustomize
    - repoURL: https://github.com/k1nho/homelab.git
      targetRevision: HEAD
      path: apps/cnpg

    - repoURL: https://cloudnative-pg.github.io/charts
      chart: cloudnative-pg
      targetRevision: 0.27.1
      helm:
        valueFiles:
          - $values/apps/cnpg/cnpg-operator-values.yaml

    - repoURL: https://github.com/k1nho/homelab.git
      targetRevision: HEAD
      ref: values

  destination:
    namespace: cnpg-system
    server: https://kubernetes.default.svc

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

---

### Creating a PostgreSQL Cluster

We are now ready to create our PostgreSQL cluster declaratively using CNPG. For this, we create the `Cluster` custom resource provided by CNPG, we will create
this cluster to serve as the database for our [linkding]() application in the next section.

```yaml {filename="apps/linkding/db.yaml"}
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: linkding-db
spec:
  instances: 3
  storage:
    # storageClass: "" # empty will use default
    size: 1Gi

  imageName: ghcr.io/cloudnative-pg/postgresql:18.3

  # Monitoring
  monitoring:
    enablePodMonitor: true

  # Bootstrap DB
  bootstrap:
    initdb:
      database: linkding
      owner: linkding
      secret:
        name: linkding-db-user-credentials
```

In this resource, we define the following:

- `instances: 3`: 3 databases should be up, one primary and two replicas.
- `size: 1Gi`: each database will have 1 Gibibyte of storage.
- `enablePodMonitor: true`: we will enable pod monitoring (useful for prometheus).
- Lastly, the bootstrap section instructs the operator to create the database with name `linkding` and owner `linkding` while also picking up the credentials for the db from a pre-created secret.

For a more comprehensive guide on all the knobs of the Cluster custom resource, check out the [examples](https://cloudnative-pg.io/docs/1.28/samples) and the official [specification](https://cloudnative-pg.io/docs/1.28/cloudnative-pg.v1#clusterspec).

### CNPG Operator Automated Backups and Recovery

An important field in the Cluster custom resource is [backup](https://cloudnative-pg.io/docs/1.28/cloudnative-pg.v1#backupconfiguration). The configuration for backup allows us to define how the backups of the cluster are taken including `volumeSnapshots` and `barmanObjectStorage`.
A simple backup configuration can be as follows:

```yaml
backup:
  barmanObjectStore:
    destinationPath: s3://BUCKET_NAME/path/to/folder
    endpointURL: http://custom-endpoint:1234
    s3Credentials:
      accessKeyId:
        name: backup-creds
        key: ACCESS_KEY_ID
      secretAccessKey:
        name: backup-creds
        key: ACCESS_SECRET_KEY
    wal:
      compression: gzip
    data:
      compression: gzip
      encryption: AES256
      immediateCheckpoint: false
      jobs: 2
  retentionPolicy: "30d"
```

In this case we define the destination path where the backups will be in our bucket and the backend endpoint url. Next, we provide our credentials to connect to it, the
compression format for write-head logs and data, and the number of days to retain the data.

The `bootstrap` field provides multiple ways to start up the database including the [recovery option](https://cloudnative-pg.io/docs/1.28/bootstrap#bootstrap-from-a-backup-recovery)
which allows us to setup [boostrap from an object storage with the barman plugin](https://cloudnative-pg.io/docs/1.28/recovery#recovery-from-an-object-store-with-the-barman-cloud-plugin). This feature allows us to recover our PostgreSQL
database from an external object storage

---

### Installing the CNPG kubectl plugin

The CNPG project provides a very nice plugin for kubectl to analyze the state of our deployed cluster and its associated databases. We will install it and work with it
in the next section. Using debian packages, we can install it as follows:

```bash
wget https://github.com/cloudnative-pg/cloudnative-pg/releases/download/v1.28.2/kubectl-cnpg_1.28.2_linux_x86_64.deb \
  --output-document kube-plugin.deb
```

```bash
sudo dpkg -i kube-plugin.deb
```

Check the appropriate installation from all the possible [installation options for the plugin](https://cloudnative-pg.io/docs/1.28/kubectl-plugin#install).

---

## Linkding Bookmark Manager

Recently, I've been doing a lot of research for the series and that includes reading documentation for the different technologies, and also blogs across the internet.
I have been bookmarking a lot, however, when it comes to retrieving that link it becomes buried, when switching from one device to another. Let's fix that, I'll install the Linkding bookmark manager, a fantastic
application service to keep track and organize my different links by topics that I can then retrieve easily, this app uses SQlite by default, but we can provide a PostgreSQL database
now that we have our CNPG operator up and running!

### Preparing the Linkding Resources

Linkding is shipped as a container image, so for this installation we will organize our different manifests and complete the install with Kustomize.

```yaml {filename="apps/linkding/deployment.yaml"}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linkding
  labels:
    app: linkding
spec:
  replicas: 1
  selector:
    matchLabels:
      app: linkding
  template:
    metadata:
      labels:
        app: linkding
    spec:
      containers:
        - name: linkding
          image: ghcr.io/sissbruecker/linkding:1.45.0-alpine
          ports:
            - containerPort: 9090
          envFrom:
            - secretRef:
                name: linkding-env-vars
          resources:
            requests:
              cpu: "50m"
              memory: "128Mi"
            limits:
              memory: "256Mi"
```

### Checking the Status of the Cluster

We will use check the general status of the cluster with the cnpg kubectl plugin:

```bash
kubectl cnpg status linkding-db -n linkding
```

```terminal
Name                     linkding/linkding-db
System ID:               4839201756618043927
PostgreSQL Image:        ghcr.io/cloudnative-pg/postgresql:18.3
Primary instance:        linkding-db-1
Primary promotion time:  2026-04-26 01:47:32 +0000 UTC (1193h27m2s)
Status:                  Cluster in healthy state
Instances:               3
Ready instances:         3
Size:                    529M
Current Write LSN:       0/20000000 (Timeline: 1 - WAL File: 000000010000000000000020)

Continuous Backup not configured

Streaming Replication status
Replication Slots Enabled
Name           Sent LSN    Write LSN   Flush LSN   Replay LSN  Write Lag  Flush Lag  Replay Lag  State      Sync State  Sync Priority  Replication Slot
----           --------    ---------   ---------   ----------  ---------  ---------  ----------  -----      ----------  -------------  ----------------
linkding-db-2  0/20000000  0/20000000  0/20000000  0/20000000  00:00:00   00:00:00   00:00:00    streaming  async       0              active
linkding-db-3  0/20000000  0/20000000  0/20000000  0/20000000  00:00:00   00:00:00   00:00:00    streaming  async       0              active

Instances status
Name           Current LSN  Replication role  Status  QoS         Manager Version  Node
----           -----------  ----------------  ------  ---         ---------------  ----
linkding-db-1  0/20000000   Primary           OK      BestEffort  1.28.1           sliffer
linkding-db-2  0/20000000   Standby (async)   OK      BestEffort  1.28.1           sliffer
linkding-db-3  0/20000000   Standby (async)   OK      BestEffort  1.28.1           sliffer
```

Interestingly enough if we check for `StatefulSet` resources:

```bash
kubectl get sts -n linkding
```

We notice that the operator itself does not create any of them. This is because CNPG manages the Kubernetes storage primitive, that is `PersistentVolumes` and `PersistentVolumeClaims`, directly.

```bash
kubectl get pvc -n linkding
```

```terminal
tbw
```

## Wrapping up

- **Previous:** [Kinho's Homelab Series - Orchestration Platform and Networking (K3s + Cilium)]({{< relref "hs_2_k3scilium/index.md">}})
- **Next:** TBD

---

## Resources

- [CloudNativePG Documentation](https://cloudnative-pg.io/docs/1.28/)
- [Fantastic Article by Gabriele Bartolini (CNPG Maintainer) on the essential commands for DBA on Kubernetes](https://www.gabrielebartolini.it/articles/2025/10/postgres-in-kubernetes-the-commands-every-dba-should-know/)
- []
