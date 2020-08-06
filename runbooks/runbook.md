# UPP - Publish Availability Monitor

For every piece of published content, polls the relevant endpoints to ascertain whether the publish was successful, within the time limit as per SLA. The resulting metrics are fed to Splunk.

## Code

publish-availability-monitor

## Primary URL

https://github.com/Financial-Times/publish-availability-monitor

## Service Tier

Platinum

## Lifecycle Stage

Production

## Delivered By

content

## Supported By

content

## Known About By

- dimitar.terziev
- elitsa.pavlova
- hristo.georgiev
- donislav.belev
- mihail.mihaylov
- boyko.boykov

## Host Platform

AWS

## Architecture

The Publish Availability Monitor listens to new content messages via the Kafka topic NativeCmsPublicationEvents, and polls (or in the case of Notifications Push listens) to the following endpoints until the content appears or the content SLA times out:

- The Content Public Read /content/{uuid} endpoint.
- The Notifications /content/notifications endpoint.
- The Notifications Push /content/notifications-push endpoint.

Additionally, if the content is an image, PAM will check the S3 image bucket
to see that the binary of the image has been saved. If two of the last ten
pieces of content failed any of these checks, then the PAM healthcheck will
switch to RED and therefore cause the cluster healthchecks and good-to-go
endpoints to show RED.

## Contains Personal Data

No

## Contains Sensitive Data

No

## Failover Architecture Type

ActiveActive

## Failover Process Type

PartiallyAutomated

## Failback Process Type

PartiallyAutomated

## Failover Details

The service is deployed in both Publishing clusters. The failover guide for the cluster is located [in the upp-docs failover guides](https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/):

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

The release is triggered by making a Github release which is then picked up by a Jenkins multibranch pipeline. The Jenkins pipeline should be manually started in order for it to deploy the helm package to the Kubernetes clusters.

## Key Management Process Type

NotApplicable

## Key Management Details

There is no key rotation procedure for this system.

## Monitoring

Pod health:

- <https://upp-prod-publish-eu.ft.com/__health/__pods-health?service-name=publish-availability-monitor>
- <https://upp-prod-publish-us.ft.com/__health/__pods-health?service-name=publish-availability-monitor>

## First Line Troubleshooting

Please DO NOT restart publish availability monitor until Second Line support have taken a look. Most errors are not caused by the service itself and it contains useful debugging information in memory. If the application is running fine, and the healthcheck is still alerting, there may be issues with the Kafka Proxy or Kafka itself. Check if these systems are up.

[Publish availability monitor panic guide](https://sites.google.com/a/ft.com/universal-publishing/ops-guides/publish-availability-monitor-panic-guide)
[First Line Troubleshooting guide](https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting)

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.
