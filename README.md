# Kpt Functions

This repository provides [Kpt functions](https://googlecontainertools.github.io/kpt/guides/producer/functions)
that are used to extend Kpt's functionality.

## Sync function

### Motivations

The sync function addresses the following shortcomings in Kpt as it exists today:
* Setters cannot be set declaratively. This is desirable, because it's nice to be able to define the values for all
of your setters in a single configuration file that can be source controlled.
* Setter cascading doesn't work (yet). Although this is planned, we need this now to enable setting common variables
at a cluster level (e.g. region, cluster name etc.), and then being able to set these values in all of the child packages.
* Some fields can not be adequately configured using kpt, in particular, fields inside of yaml literals. This is crucial
for some applications such as setting inline policies inside of IRSA roles, or setting Cloudwatch configuration JSON.

### Usage

Suppose that the target repository has the following structure.

```
.
├── config
    ├── development
    │   └── ap-southeast-2
    │       └── a
    │           └── packages.yaml
    └── production
        └── ap-southeast-2
            └── a
                └── packages.yaml
```

Below shows what `config/development/ap-southeast-2/a/packages.yaml` looks like.

```yaml
apiVersion: kpt.seek.com/v1alpha1
kind: ClusterPackages
metadata:
  name: ap-southeast-2-development-a
spec:
  baseDir: config/development/ap-southeast-2/a/packages
  packages:
  - name: cert-manager
    git:
      repo: git@github.com:seek-oss/packages.git
      directory: cert-manager
      ref: 5fc702d3dd0f46509283cb0bcc4a3327d1ee8b1d
      # Packages may define their own overrides
      #variables:
      #- name: foo
      #  value: bar
  - name: istio-system
    git:
      repo: git@github.com:seek-oss/packages.git
      directory: istio-system
      ref: 5fc702d3dd0f46509283cb0bcc4a3327d1ee8b1d
  variables:
  - name: account-id
    value: "1234"
  - name: region
    value: ap-southeast-2
  - name: region-short
    value: apse2
  - name: cluster
    value: development-a
  - name: environment
    value: development
  - name: oidc-provider-id
    value: ABCDEFG
  - name: internal-domain-name
    value: ap-southeast-2.development.internal.seek.com
  - name: external-domain-name
    value: ap-southeast-2.development.external.seek.com
  - name: internal-hosted-zone-id
    value: INT001
  - name: external-hosted-zone-id
    value: EXT001
  - name: irsa-policy
    value: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Principal": {
              "Federated": 'arn:aws:iam::{{value "account-id"}}:oidc-provider/oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}'
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
              "StringEquals": {
                "oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}:sub": "system:serviceaccount:{{args 0}}:{{args 1}}"
              }
            }
          }
        ]
      }
```

Then the following operation would sync all packages, and render any templates.

```bash
kpt fn source \
  config/development/ap-southeast-2/a/packages.yaml \
  config/production/ap-southeast-2/a/packages.yaml \
  | kpt fn run \
  --image docker.io/seek/kpt-sync:latest \
  --mount type=bind,source="${HOME}/.ssh/id_rsa,target=/.ssh/id_rsa,readonly" \
  --network -- logLevel=debug \
  | kpt fn sink .
```

### Authentication

In order to clone repositories that are private, kpt needs ssh credentials

#### Mounting a key file

You can mount an identity file from your `.ssh` directory. However this requires that you have no passphrase on
the ssh key.

This can be accomplished by passing `--mount type=bind,source="${HOME}/.ssh/id_rsa,target=/.ssh/id_rsa,readonly"` to
`kpt fn run`

#### SSH agent forwarding

If you are running an SSH agent on your host, you can forward this host into your Docker container to authenticate
using keys already loaded into its keyring.

Pass the following to `kpt fn run`

```
--mount type=bind,src="/run/host-services/ssh-auth.sock",target="/run/host-services/ssh-auth.sock" \
-e SSH_AUTH_SOCK="/run/host-services/ssh-auth.sock"
```

#### Troubleshooting agent forwarding

SSH agent forwarding between OSX and docker containers seems to be incredibly complicated and hard to get right.
There are numerous threads that show seemingly simple fixes that don't seem to work for others.

What worked for me:

```shell
$ docker --version
Docker version 20.10.5, build 55c4c88
Docker desktop version 3.2.2 (61853)
$ uname -rvmi
19.6.0 Darwin Kernel Version 19.6.0: Tue Jan 12 22:13:05 PST 2021; root:xnu-6153.141.16~1/RELEASE_X86_64 x86_64 MacBookPro16,1
```

The following arguments are required for forwarding to work. Apparently this path is a special path that isn't
actually on the host, it's inside the VM that Docker Desktop on mac uses.

```
  --mount type=bind,src="/run/host-services/ssh-auth.sock",target="/run/host-services/ssh-auth.sock" \
  -e SSH_AUTH_SOCK="/run/host-services/ssh-auth.sock"
```

This socket is owned by root, so trying to use it by default will fail if your Dockerised application isn't running
as root. To work around this, run the following from your host machine:

```
docker run -it --privileged --pid=host debian nsenter -t 1 -m -u -n -i sh -c 'chmod o+w /run/host-services/ssh-auth.sock'
```

This didn't work one day, and then did the next day. I don't know why.

## Hash dependency function

### Motivations

Some Kubernetes applications read their config from a config map or some other configuration source, but are not
configured to automatically reload this configuration when it changes. The solution to this is to restart the
workload, but this must be done manually.

The hash dependency function allows for embedding a hash of a dependant piece of configuration in a workload spec.
When the configuration changes, the workload will necessarily be re-created because of the changed
annotation in its spec.

See [cmd/hash-dependency/README.md](cmd/hash-dependency/README.md) for more info.
