# Deploying grafiti as a Kubernetes CronJob

[Kubernetes][kubernetes-docs] [CronJobs][kubernetes-docs-cronjob] schedule programs to run periodically or at a given point in time. Deploying grafiti to a Kubernetes allows you to clean your AWS account periodically, and aggregate and forward deletion logs. Creating and managing a grafiti CronJob can be made even easier using [Tectonic][tectonic-website], CoreOS' self-driving Kubernetes software.

## Setting up a CronJob

1. Create a Kubernetes [CronJob config file][kubernetes-docs-cronjob-config]. Ensure container environments are provisioned with the following:
    * Valid AWS credentials (environment variables or a 'credentials' file)
    * A grafiti configuration file and/or environment variables
    * Data or tag input files, depending on which sub-command you are running

Example CronJob configuration file:
```yaml
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: grafiti-deleter
spec:
  schedule: "* */6 * * *" # Run every 6 hours
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - command:
            - /bin/bash
            - -c
            - grafiti -e -c /opt/config.toml delete --all-deps -f /opt/tags.json
            env:
              # Specify GRF_* and AWS_* environment variables here
              - name: AWS_REGION
                value: us-east-1
            name: grafiti-deleter
            image: your/registry/grafiti:v0.1.1
            volumeMounts:
              # Mount a set of AWS credentials. Alternatively, add your own secret:
              # https://kubernetes.io/docs/concepts/configuration/secret/
              - mountPath: /root/.aws/credentials
                name: grafiti-aws-credentials
                readOnly: true
              - mountPath: /opt/config.toml
                name: config-path
              - mountPath: /opt/tags.json
                name: tags-path
            securityContext:
              runAsNonRoot: true
              runAsUser: 1000
          volumes:
            - hostPath:
                path: ~/.aws/credentials # Specify location of AWS credentials you want to mount
              name: grafiti-aws-credentials
            - hostPath:
                path: ./config.toml # Add your own config file path here
              name: config-path
            - hostPath:
                path: ./example-tags-input.json # Add your own tag file path here
              name: tags-path
          restartPolicy: OnFailure
```

2. Run your Kubernetes API server with the `--runtime-config=batch/v2alpha1=true` flag to enable the CronJob API version. If you're using Tectonic, navigate to Console -> Workloads -> Daemon Sets -> YAML tab, and add a `- --runtime-config=batch/v2alpha1=true` field under the `containers.name:kube-apiserver.command` section.
    * **Note**: updates to Kubernetes may cause this flag to be reset (see the Kubernetes [API versioning docs][kubernetes-docs-api-versioning] for more information on enabling API versions). Tectonic does not recommend using non-default manifest file flags at the moment, but will support persistent changes to manifest files soon.

3. Restart your API server. If you're using Tectonic, your API server pod will reload itself after clicking 'Save Changes'.

4. [Set up and configure][kubernetes-docs-tutorials-kubectl] `kubectl`.

5. Follow the Kubernetes [documentation][kubernetes-docs-cronjob] to create your CronJob using `kubectl`. You're all set!

Further Kubernetes documentation:
 * Creating a [cluster][kubernetes-docs-aws] in AWS
 * [kubectl cheatsheet][kubernetes-docs-kubectl-cheatsheet]
 * Creating a [secret][kubernetes-docs-config-secret]

Further Tectonic documentation:
 * Creating a [cluster][tectonic-docs-aws] in AWS
 * Deploying an [application][tectonic-docs-tutorials-deploy-app] on your cluster

## Logging

The Kubernetes [logging architecture][kubernetes-docs-logging], which uses [fluentd][fluentd-website] as its logging layer, can aggregate and forward log data from log files to an endpoint of your choices, like an S3 bucket. More information on grafiti's logging capabilities can be found in our [usage notes][grafiti-usage-notes].

[grafiti-usage-notes]: [usage-notes-and-tips.md#logging]

[fluentd-website]: http://www.fluentd.org/

[kubernetes-docs]: https://kubernetes.io/docs/home/
[kubernetes-docs-aws]: https://kubernetes.io/docs/getting-started-guides/aws/
[kubernetes-docs-api-versioning]: https://kubernetes.io/docs/concepts/overview/kubernetes-api/#enabling-api-groups
[kubernetes-docs-config-secret]: https://kubernetes.io/docs/concepts/configuration/secret/
[kubernetes-docs-cronjob]: https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/
[kubernetes-docs-cronjob-config]: https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#creating-a-cron-job
[kubernetes-docs-kubectl-cheatsheet]: https://kubernetes.io/docs/user-guide/kubectl-cheatsheet/
[kubernetes-docs-logging]: https://kubernetes.io/docs/concepts/cluster-administration/logging/
[kubernetes-docs-tutorials-kubectl]: https://coreos.com/tectonic/docs/latest/tutorials/first-app.html

[tectonic-website]: https://coreos.com/tectonic/
[tectonic-docs-aws]: https://coreos.com/tectonic/docs/latest/install/aws/index.html
[tectonic-docs-tutorials-deploy-app]: https://coreos.com/tectonic/docs/latest/tutorials/first-app.html
