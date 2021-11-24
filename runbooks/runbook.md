<!--
    Written in the format prescribed by https://github.com/Financial-Times/runbook.md.
    Any future edits should abide by this format.
-->
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

## Host Platform

AWS

## Architecture

The Publish Availability Monitor listens to new content messages via the Kafka topic NativeCmsPublicationEvents, and polls (or in the case of Notifications Push listens) to the following endpoints until the content appears or the content SLA times out:

*   The Content Public Read /content/{uuid} endpoint.
*   The Content Notifications /content/notifications endpoint.
*   The Content Notifications Push /content/notifications-push endpoint.

*   The Pages Public Read /__public-pages-api/pages/{uuid} endpoint.

*   The Lists Public Read /__public-lists-api/lists/{uuid} endpoint.
*   The Lists Notifications /__list-notifications-rw/lists/notifications endpoint.
*   The Lists Notification Push /lists/notifications-push endpoint.

If two of the last ten pieces of content failed any of these checks, then the PAM healthcheck will
switch to RED and therefore cause the cluster healthchecks and good-to-go
endpoints to show RED.

## Contains Personal Data

No

## Contains Sensitive Data

No

<!-- Placeholder - remove HTML comment markers to activate
## Can Download Personal Data
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

<!-- Placeholder - remove HTML comment markers to activate
## Can Contact Individuals
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

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

<!-- Placeholder - remove HTML comment markers to activate
## Heroku Pipeline Name
Enter descriptive text satisfying the following:
This is the name of the Heroku pipeline for this system. If you don't have a pipeline, this is the name of the app in Heroku. A pipeline is a group of Heroku apps that share the same codebase where each app in a pipeline represents the different stages in a continuous delivery workflow, i.e. staging, production.

...or delete this placeholder if not applicable to this system
-->

## Key Management Process Type

NotApplicable

## Key Management Details

There is no key rotation procedure for this system.

## Monitoring

Pod health:

*   <https://upp-prod-publish-eu.ft.com/__health/__pods-health?service-name=publish-availability-monitor>
*   <https://upp-prod-publish-us.ft.com/__health/__pods-health?service-name=publish-availability-monitor>

## First Line Troubleshooting

Please DO NOT restart publish availability monitor until Second Line support have taken a look. Most errors are not caused by the service itself and it contains useful debugging information in memory. If the application is running fine, and the healthcheck is still alerting, there may be issues with the Kafka Proxy or Kafka itself. Check if these systems are up.

[Publish availability monitor panic guide](https://sites.google.com/a/ft.com/universal-publishing/ops-guides/publish-availability-monitor-panic-guide)
[First Line Troubleshooting guide](https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting)

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.
