# Kpt tutorial

This tutorial serves as an introduction to the two main Kpt workflows: package authoring and package consumption.
Understanding these two workflows will give you an appreciation for Kpt's utility as a Kubernetes resource management
system.

## 1. Kpt setup

Follow the [Kpt installation documentation](https://googlecontainertools.github.io/kpt/installation/) to install Kpt.

## 2. Consuming a package

Before we _author_ Kpt packages, we will _consume_ Kpt packages.
This will show us what advantages Kpt has over other resource management tools.

The first step in consuming a Kpt package is to `get` it.

In an empty directory, run
```bash
$ kpt pkg get https://github.com/seek-oss/kpt-functions/examples/tutorial/sample-package tutorial-package
fetching package "examples/tutorial/sample-package" from "https://github.com/seek-oss/kpt-functions" to "tutorial-package"
```
This fetches a Kpt package from the specified Git repository into the local directory `tutorial-package`.
Let's take a look at the files we got:
* `Kptfile`: this is the main configuration file used by Kpt.
* `deployment.yaml`: this is a standard Kubernetes manifest file, but with some annotations added by Kpt.
* `README.md`

This sample Kpt package deploys a deployment of the httpbin application.
However because it is a Kpt package, we can use the Kpt CLI to customize some aspects of this application.
The options we have for customization are:
* Customizing the `owner` metadata of the application
* Customizing the number of replicas that are deployed
* Customizing the image tag of httpbin that is deployed
We can discover the values available to us for customization by running
```bash
$ kpt cfg list-setters tutorial-package
tutorial-package/
    NAME           VALUE         SET BY   DESCRIPTION   COUNT   REQUIRED   IS SET
  image-tag   latest                                    1       No         Yes
  owner       my-team                                   2       No         Yes
  replicas    2                                         1       No         Yes
```

In order to set these values, we can run the following
```bash
$ kpt cfg set tutorial-package replicas 5
sample-package/
set 1 field(s) of setter "replicas" to value "5"
```

Observe that Kpt has updated the `Kptfile` as well as the `deployment.yaml` to reflect your setting of the values.
You can set the other setters similarly.

Because Kpt uses annotations inside of YAML comments, the `deployment.yaml` file can be deployed as is.

## 3. Authoring packages

Now we will learn how to create our own Kpt packages.
We will be re-creating the Kpt package we consumed in the previous section.

Create a git repository that you can publish somewhere. The repository can be public or private, as long as v0.19.4
have permission to pull and push from/to it.
In the root directory of the repository, run:
```bash
$ mkdir sample-package
$ kpt pkg init sample-package
writing "sample-package/Kptfile"
writing "sample-package/README.md"
```
This will create a directory and initialise a Kpt package in that directory.
Note that Kpt creates a `Kptfile` for you, as well as a `README.md` with instructions for using this Kpt package.
Add a Kubernetes manifest for specifying a deployment at `sample-package/deployment.yaml`. A sample is provided below,
or use your own.
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-deploy
  namespace: sample
  labels:
    app: sample-deploy
    owner: my-team
spec:
  replicas: 2
  selector:
    matchLabels:
      app: sample-deploy
  template:
    metadata:
      labels:
        app: sample-deploy
        owner: my-team
    spec:
      containers:
      - name: httpbin
        image: kennethreitz/httpbin:latest
```

We want to allow consumers of this Kpt package to configure the number of replicas of this application themselves.
This requires us to use a Kpt setter.
To add a setter, run
```bash
$ kpt cfg create-setter sample-package replicas 2
sample-package/
created setter "replicas"
```
The syntax here is:
```
kpt cfg create-setter <directory> <setter-name> <setter-value>
```
This creates a setter called `<setter-name>` in the Kpt package at `<directory>`.
If Kpt finds any values that match `<setter-value>` in your YAML, it will automatically link those values to the
setter you create.
In our case, Kpt will find the `replicas: 2` value, and automatically add the annotations to this line to link
it to the setter we just created.
This annotation looks like this: `# {"$kpt-set":"replicas"}`.
Look at your `deployment.yaml` file and make sure that you can see this annotation added on the `replicas` line.
Kpt stores information about setters in the `Kptfile`.
Your `Kptfile` should now look like this:
```yaml
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: sample-package
packageMetadata:
  shortDescription: sample description
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: replicas
          value: "2"
```
Note that the `name` field of the `setter` matches the annotation that Kpt added to your source file above, and
that the `value` field has been populated for you.

Consumers of this package can now modify the `replicas` field by running
```bash
$ kpt cfg set sample-package replicas 5
sample-package/
set 1 field(s) of setter "replicas" to value "5"
```
Check that this works, and check that both the `Kptfile` and `deployment.yaml` are updated correctly with the set
value.

Let's add another setter to this package.
This time, we'll add a setter that is used in multiple places.
Run
```bash
$ kpt cfg create-setter sample-package owner my-team
sample-package/
created setter "owner"
```
Note that Kpt has updated your `deployment.yaml` and your `Kptfile` again.
Try setting this value with
```bash
$ kpt cfg set sample-package owner a-different-team
sample-package/
set 2 field(s) of setter "owner" to value "a-different-team"
```
Check that this has updated correctly

For our final piece of Kpt configuration, we are going to use a substitution.
Substitutions allow us insert a setter into a larger string.
In our example, we'll use this to set the value of the Docker image tag.
Run
```bash
$ kpt cfg create-subst sample-package image --field-value kennethreitz/httpbin:latest --pattern kennethreitz/httpbin:\${image-tag}
sample-package/
unable to find setter with name image-tag, creating new setter with value latest
created substitution "image"
```
The syntax for this is:
```
kpt cfg create-subst <directory> <substitution-name> --field-value <value-of-final-substitution> --pattern <pattern-for-substitution>
```
The `<value-of-final-substitution>` is similar to the `<setter-value>` field above.
Kpt will look for values that match this, and use them to automatically create the required annotations in your
source file, and to set the value for this substitution in your `Kptfile`.
The `<pattern-for-substitution>` field is where you define how to take setters and insert them into your substitution
value.
These setters are declared using the `\${<setter-name}` syntax.
These setters need not exist prior to declaring the substitution, as was the case in the command above.
Kpt create a setter for us, and automatically assigned it a value based on the pattern and the field value.

We don't use substitutions directly.
Kpt takes care of updating a substitution whenever a setter changes.
Set the value of the `image-tag` setter to something using
```bash
$ kpt cfg set sample-package image-tag 1.2.3
sample-package/
set 1 field(s) of setter "image-tag" to value "1.2.3"
```
Check the `Kptfile` and the `deployment.yaml` to see that the value of the setter and the image were correctly updated.

You can now commit and push this Kpt package to your Git repository.
You should be able to consume it in an identical way to the way we consume the sample package above, substituting
the URL to the Kpt package for the URL to your own Git repository.
