---
order: 4
---

# Systemctl and Services

`systemd` is a system and service manager for Linux operating systems, designed
for the management and configuration of "services", which are programs that run
in the background (such as Nibiru nodes)  without user intervention.
`systemctl` is a command-line tool that interacts with `systemd`. {synopsis}

TLDR: [Quick Start: Systemctl for Running Nibiru Nodes](#quick-start-systemctl-for-running-nibiru-nodes)

::: tip
It's recommended to use either [Cosmovisor](./cosmovisor) or `systemctl` to run
your node.
:::

## What is a service?

In essence, a "service" in this context is like an "app" on your phone that
runs in the background, checking for updates or notifications, without you
having to manually open it.

Services can range from simple daemons that log system activity to complex
software like web servers or databases. Systems are distinct from
user-interactive applications that you might launch from a desktop environment.

In the realm of Linux and Unix-like operating systems, these background
processes have traditionally been called **"daemons"**, hence the "d" at the end of
many service names (e.g. systemd, nibid).

## Systemctl

The `systemctl` command-line tool is a crucial component provided by `systemd`.
It allows users to introspect and control the state of the `systemd` system and
service manager. With `systemctl`, operations like starting, halting, or
rebooting services become straightforward.

## Understanding Service Units

Within `systemd`, each **service** is overseen by a
**service unit**. This unit contains configuration details that dictate how the
service starts, stops, and operates during its runtime.

The service unit defines things like
what command to run to start the service, what files or other services it
depends on, and various other parameters.

By defining and controlling these services, `systemd` allows for:

1. **Parallelization:** Services can be started concurrently during the boot
   process to reduce boot time.
2. **Dependency Handling:** Services can be set to start only after certain
   conditions are met, like other services starting first.
3. **Automatic Recovery:** If a service crashes, `systemd` can be configured to
   restart it.
4. **Resource Control:** Set limits on the amount of
   resources (CPU, memory) a service can use.

## Quick Start: Systemctl for Running Nibiru Nodes

::: tip
If you have not installed `nibid`, please install the required version of the binary (e.g. vX.Y.Z) using the following command:

```bash
curl -s https://get.nibiru.fi/@vX.Y.Z! | bash
```
:::

1. Create a service file.

    Executing the following command will create a service definition for
    `nibid` in the directory for systemd configuration (`/etc/systemd/system`).

    Be sure to fill in the placeholder variables like `<your_user>` and
    `<your_user_group>` before copying the command. If you're which value to
    put as your "user", it's the value returned by running `whoami` in the
    terminal.

    ```bash
    sudo tee /etc/systemd/system/nibiru.service<<EOF
    [Unit]
    Description=Nibiru Node
    Requires=network-online.target
    After=network-online.target

    [Service]
    Type=exec
    User=<your_user>
    Group=<your_user_group>
    ExecStart=/usr/local/bin/nibid start --home /home/<your_user>/.nibid
    Restart=on-failure
    ExecReload=/bin/kill -HUP $MAINPID
    KillSignal=SIGTERM
    PermissionsStartOnly=true
    LimitNOFILE=65535

    [Install]
    WantedBy=multi-user.target
    EOF
    ```

2. Enable the service

    This makes sure that the service starts automatically.

    ```bash
    # Refresh systemd's knowledge of available services
    sudo systemctl daemon-reload

    # Enable the `nibid` service.
    sudo systemctl enable nibiru
    ```
