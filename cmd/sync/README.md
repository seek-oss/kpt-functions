# Kpt Sync Function

A function to declaratively sync multiple Kpt packages that share configuration.

## Examples

* [Basic example](./examples/sync/basic/README.md)

## Basic usage

Define a `ClusterPackages` file:

```yaml
# config/development/ap-southeast-2/a/packages.yaml
apiVersion: kpt.seek.com/v1alpha1
kind: ClusterPackages
metadata:
  name: development-a-ap-southeast-2
spec:
  baseDir: config/development/ap-southeast-2/a/packages
  packages:
  - name: some-application
    git:
      repo: git@github.com:seek-oss/packages.git
      directory: some-application
      ref: 5fc702d3dd0f46509283cb0bcc4a3327d1ee8b1
      variables:
      - name: foo
        value: bar
  variables:
  - name: account-id
    value: "1234"
  - name: region
    value: ap-southeast-2
  - name: cluster
    value: development-a
```

To perform a sync, run the command below:

```bash
kpt fn source \
  config/development/ap-southeast-2/a/packages.yaml \
  | kpt fn run \
  --image docker.io/seek/kpt-sync:latest \
  --network -- logLevel=debug \
  | kpt fn sink .
```

The packages defined at `seek-oss/packages` will be rendered to the `config/development/ap-southeast-2/a/packages`
directory.

Setters with names that match the variables defined in your `packages.yaml` will be set to their appropriate values,
with variables that are defined under the package taking precedence over global variables.

## Motivations

The sync function addresses the following shortcomings in Kpt as it exists today:
* Setters cannot be set declaratively. We would like a mechanism to create the Kpt equivalent of a Helm `values.yaml`
  file which contains the desired values for all of the setters for a Kpt package.
* Lack of a higher order construct that can be used to manage separate but related Kpt packages. This is fairly common
  when managing a Kubernetes cluster. In Kpt language, you have a number of Kpt packages that map to individual
  applications, and these applications share common setters such as `region` or `cluster`. This seems to be a goal of
  the Kpt project through cascading of setter values to subpackages, but is either unimplemented or not-ready for
  use at this stage.
* Some fields can not be adequately configured using Kpt, in particular, fields inside of yaml literals. This is crucial
  for some applications such as setting inline policies inside of IRSA roles, or setting Cloudwatch configuration JSON.

## Authentication

In order to clone repositories that are private, Kpt needs SSH credentials.

### Mounting a key file

You can mount an identity file from your `.ssh` directory. However this requires that you have no passphrase on
the ssh key, which is not recommended.

This can be accomplished by passing `--mount type=bind,source="${HOME}/.ssh/id_rsa,target=/.ssh/id_rsa,readonly"` to
`kpt fn run`

### SSH agent forwarding

If you are running an SSH agent on your host, you can forward this host into your Docker container to authenticate
using keys already loaded into its keyring.

Pass the following to `kpt fn run`

```
--mount type=bind,src="/run/host-services/ssh-auth.sock",target="/run/host-services/ssh-auth.sock" \
-e SSH_AUTH_SOCK="/run/host-services/ssh-auth.sock"
```

### Troubleshooting agent forwarding

SSH agent forwarding between OSX and Docker containers seems to be incredibly complicated and hard to get right.
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


## Advanced usage

### Syncing multiple clusters at the same time

The sync function caches Git repos internally. This significantly speeds up execution time if you are syncing multiple
clusters with packages from the same repo. The sync function uses the `spec.baseDir` field to ensure that the output
keeps input resources separate.

Consider the following set of sync `packages.yaml` files

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

Then the following operation would sync all packages and output them back into their original directories.

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

Note: you will need to clear the directories declared as `spec.baseDir`s yourself before running the above.

### Templating

The sync function can render templates inside of package files. This is useful for situations where Kpt cannot be used
to set a value because the values is inside of a YAML multiline string.

In your Kpt package:

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: some-name
  namespace: some-namespace
data:
  # {"$kpt-template":"true"}
  my-data.json: |
    {{render "my-template" "foo" "bar"}}
```

**Important!**: Make sure you have a setter configured in your Kpt package with the name matching the name of the template

In your `packages.yaml`:

```yaml
# config/development/ap-southeast-2/a/packages.yaml
apiVersion: kpt.seek.com/v1alpha1
kind: ClusterPackages
metadata:
  name: development-a-ap-southeast-2
spec:
  baseDir: config/development/ap-southeast-2/a/packages
  packages:
  - name: some-application
    git:
      repo: git@github.com:seek-oss/packages.git
      directory: some-application
      ref: 5fc702d3dd0f46509283cb0bcc4a3327d1ee8b1
  variables:
  - name: region
    value: ap-southeast-2
  - name: my-template
    value: |
      {
        "region": "{{value "region"}}",
        "first-arg": "{{args 0}}",
        "second-arg": "{{args 1}}"
      }
```

After running a sync, your configmap will be rendered as:

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: some-name
  namespace: some-namespace
data:
  # {"$kpt-template":"true"}
  my-data.json: |
    {
      "region": "ap-southeast-2",
      "first-arg": "foo",
      "second-arg": "bar"
    }
```

Note: It's currently not possible to use Kpt substitutions in templates. This issue is tracked as [issue #15](https://github.com/seek-oss/kpt-functions/issues/15)
