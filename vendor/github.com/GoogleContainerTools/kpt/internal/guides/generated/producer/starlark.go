// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by "mdtogo"; DO NOT EDIT.
package producer

var StarlarkGuide = `
{{% pageinfo color="warning" %}}
The Starlark runtime is Alpha. It is disabled by default, and must be enabled
with the ` + "`" + `--enable-star` + "`" + ` flag.
{{% /pageinfo %}}

Functions may be written as [Starlark] scripts which modify a ResourceList
provided as a variable.

#### Imperative Run

Starlark functions can be run imperatively by providing the Starlark script as
a flag on ` + "`" + `kpt fn run` + "`" + `. Following is an example of a Starlark function which
adds a "foo" annotation to every resource in the package.

  # c.star
  # set the foo annotation on each resource
  def run(r, an):
    for resource in r:
      # mutate the resource
      resource["metadata"]["annotations"]["foo"] = an
  
  # get the value of the annotation to add
  an = ctx.resource_list["functionConfig"]["data"]["value"]
  
  run(ctx.resource_list["items"], an)

Run the Starlark function with:

  # run c.star as a function, generating a ConfigMap with value=bar as the
  # functionConfig
  kpt fn run . --enable-star --star-path c.star -- value=bar

Any resource under ` + "`" + `.` + "`" + ` will have the ` + "`" + `foo: bar` + "`" + ` annotation added.

#### Declarative Run

Starlark functions can also be run declaratively using the
` + "`" + `config.kubernetes.io/function` + "`" + ` annotation. This annotation indicates that the
resource is functionConfig that should be provided to a function.

Following is an example of a Starlark function which adds a "foo" annotation to
each resource in its package. The ExampleKind resource will be set as the
ResourceList.functionConfig.

  # example.yaml
  apiVersion: example.com/v1beta1
  kind: ExampleKind
  metadata:
    name: function-input
    annotations:
      config.kubernetes.io/function: |
        starlark: {path: c.star, name: example-name}
  data:
    value: "hello world"

Example Starlark function to which will add an annotation to each resource
scoped to ` + "`" + `example.yaml` + "`" + ` (those under the directory containing ` + "`" + `example.yaml` + "`" + `):

  # c.star
  # set the foo annotation on each resource
  def run(r, an):
    for resource in r:
      resource["metadata"]["annotations"]["foo"] = an
  
  an = ctx.resource_list["functionConfig"]["data"]["value"]
  run(ctx.resource_list["items"], an)

Run them on the directory containing ` + "`" + `example.yaml` + "`" + ` using:

  kpt fn run DIR/ --enable-star

## Debugging Functions

It is possible to debug Starlark functions using ` + "`" + `print` + "`" + `

  # c.star
  print(ctx.resource_list["items"][0]["metadata"]["name"])

  kpt fn run . --enable-star --star-path c.star

> foo

## OpenAPI

The OpenAPI known to kpt is provided to the Starlark program through the
` + "`" + `ctx.open_api` + "`" + ` variable. This may contain metadata about the resources and
their types.

  #c.star
  print(ctx.open_api["definitions"]["io.k8s.api.apps.v1.Deployment"]["description"])

  kpt fn run . --enable-star --star-path c.star

> Deployment enables declarative updates for Pods and ReplicaSets.

## Retaining YAML Comments

While Starlark programs are unable to retain comments on resources, kpt will
attempt to retain comments by copying them from the function inputs to the
function outputs.

It is not possible at this time to add, modify or delete comments from
Starlark scripts.

## Next Steps

- Explore other ways to run functions from the [Functions Developer Guide].
- Find out how to structure a pipeline of functions from the
  [functions concepts] page.
- Consult the [fn command reference].
`
