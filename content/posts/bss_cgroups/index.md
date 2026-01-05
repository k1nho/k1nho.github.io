---
title: "Byte Size Series - Limiting CPU and Memory Usage with cgroups"
pubDate: 2025-11-03
Description: "Kubernetes enables the user to set limits for resources such as CPU and memory, but how is the sausage made? In this byte size series entry we go over the fundamental concept of cgroups!"
Categories: ["Linux", "DevOps", "Platform Engineering", "Byte Wisdom"]
Tags: ["Byte Wisdom", "cgroups", "Linux", "DevOps", "Learning"]
cover: "gallery/bss_cgroups_cover.png"
mermaid: true
---

The rise of containers and their adoption across the industry with technologies like **Docker and Kubernetes** has been remarkable, and with the advent of AI, these technologies are poised to become fundamental not only for training and inference of AI models but also for hosting AI services themselves.
Beneath these abstractions lies the fundamental idea that allows us to start a Linux process while **limiting its CPU and memory resources** using [**Control Groups**](https://man7.org/linux/man-pages/man7/cgroups.7.html), also know as **cgroups**. In this entry of the byte size series, we'll explore how we can use [systemd](https://systemd.io/),
one of the most used init systems in many Linux distros, to configure and add a process to a cgroup.

## What is a cgroup ?

From the [man pages](https://man7.org/linux/man-pages/man7/cgroups.7.html), we get the following:

> Control groups, usually referred to as cgroups, are a Linux kernel
> feature which allow processes to be organized into hierarchical
> groups whose usage of various types of resources can then be
> limited and monitored. The kernel's cgroup interface is provided
> through a pseudo-filesystem called cgroupfs. Grouping is
> implemented in the core cgroup kernel code, while resource
> tracking and limits are implemented in a set of per-resource-type
> subsystems (memory, CPU, and so on).

From this definition, we see that interacting with the Cgroup interface is done via the `pseudo-filesystem cgroupfs`, usually mounted at **/sys/fs/cgroup**.
To create a Cgroup manually, we would simply create a subdirectory, which then gets populated with files used to manipulate the Cgroup configuration.

Let’s move one level above this low-level abstraction and use **systemd** to control a Cgroup, which makes the work much easier.
In fact, **systemd** performs operations under the `pseudo-filesystem` as if you were using it through shell commands directly. For example, creating a Cgroup manually would involve running `mkdir` on **/sys/fs/cgroup**.

### Adding a Process to a cgroup with Systemd (Transient Setup)

To demonstrate cgroups, we will use `spin_loop.py` this is a simple program that loops forever adding more memory on each iteration.

```python {filename="spin_loop.py"}
data = []
while True:
    data.append([0] * 100)
```

Let us now add this process to a Cgroup via `systemd-run`

```bash
systemd-run -u eatmem -p CPUQuota=20% -p MemoryMax=1G python ~/spin_loop.py
```

In the command above:

- We create a **systemd unit** called eatmem.
- We set the **CPU quota to 20% and max memory usage to 1 GB**.
- Finally, we specify that we want to add the **spin_loop.py** process to the cgroup.

Whenever the program exceeds these limits (as it will in this example by consuming memory indefinitely), the **out-of-memory killer** is triggered. By running the following command, we can confirm the status of our unit:

```bash
systemd status eatmem.service
```

```plaintext
Loaded: loaded (/run/systemd/transient/eatmem.service; transient)
Transient: yes
Active: failed (Result: oom-kill)
...
```

<details>
<summary>Garbage collecting the unit</summary>

By default, **systemd does not cleanup the transient unit**, so if you run the command once again you will see.

> [!CAUTION] Error
> Failed to start transient service unit: **Unit eatmem.service was already loaded or has a fragment file.**

This default behavior is what enables us to inspect logs, and status afterwards. If you would like to manually cleanup the unit, you can run:

```bash
sudo systemctl reset-failed eatmem.service
```

If you would like systemd to automatically **garbage-collect** the transient unit, run the command with the `--collect` flag:

```bash
systemd-run --collect -u eatmem -p CPUQuota=20% -p MemoryMax=1G python ~/spin_loop.py
```

</details>

Just like that, we have prevented the process from going wild. Very cool!

### Configuring a cgroup (Persistent Setup)

There is one more thing, if we ever wanted to save this configuration, that is a cgroup that monitors and limits the CPU quota and memory max to those that we specify, we would need to define a `slice`
as otherwise the cgroup would be tied to the process that was invoked in it. We can achieve that with the following.

```plaintext {filename="sliceconfig"}
[Slice]
CPUQuota=20%
MemoryMax=1G
```

```bash
sudo cp sliceconfig /etc/systemd/system/eatmem.slice
```

Now we can place any process into our `eatmem.slice` as we did before.

```bash
systemd-run -u eatmem --slice=eatmem.slice python ~/spin_loop.py
```

If we add more processes, their **cumulative CPU and memory usage** will be limited according to our slice configuration.

## Byte Bye

**cgroups** underpin one of the most powerful and important technologies in containerization. Managing `cgroupfs` via **systemd** is a lower-level abstraction, albeit important to understand, as container managers like [Containerd](https://containerd.io/)
leverage **systemd** as a `cgroup driver` to control container resource usage. This ensures the system is protected from resource-hungry processes while distributing compute fairly.

Did you know about **cgroups**?
