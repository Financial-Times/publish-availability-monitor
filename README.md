# publish-availability-monitor

[![Circle CI](https://circleci.com/gh/Financial-Times/publish-availability-monitor/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/publish-availability-monitor/tree/master)

Monitors publish availability and collects related metrics. Collected metrics are sent to various systems (ex. Splunk).

# Usage
`go get github.com/Financial-Times/publish-availability-monitor`

`publish-availability-monitor -config=int-config.json`

With Docker:

`docker build -t coco/publish-availability-monitor .`

`docker run -ti --env QUEUE_ADDR=<addr> --env URLS=<endpoint1>,<endpoint2> coco/publish-availability-monitor`

# Puppet module
If you want a new puppet module version:
* manually increment the version in `puppet/ft-publish_availability_monitor/Modulefile`
* create a `git tag <version>` corresponding to the module version
The preferrable way for the second step is to create a github release with the module version (this will automatically create the git tag, as well).

# Build and deploy
### UCS
* prepare a new puppet module (follow the steps above)
* build puppet module using the Jenkins job `publish-availability-monitor-build`
* deploy the puppet module using the Jenkins job `publish-availability-monitor-deploy`

# Configuration

```
//this is the SLA for content publish, in seconds
//the app will check for content availability until this threshold is reached
//or the content was found  
"threshold": 120,
```

```
//Configuration for the queue we will read from
"queueConfig": {
	//URL(s) of the messaging system
	"address": [queue address1, queue address 2],
	//The group name by which this app will be knows to the messaging system
	"group": "YourGroupName",
	//The topic we want to get messages from
	"topic": "YourTopic",
	//the name of the queue we want to use
	"queue": "yourQueue"
},
```

```
//for each endpoint we want to check content against, we need a metricConfig entry
"metricConfig": [
{
	//the URL of the endpoint
	//to check content, the UUID is appended at the end of this URL
	//should end with /
	"endpoint": "endpointURL",
	//used to associate endpoint-specific behavior with the endpoint
	//each alias should have an entry in the endpointSpecificChecks map
	"alias": "content",
	//defines how often we check this endpoint
	//the check interval is threshold / granularity
	//in this case, 120 / 40 = 3 -> we check every 3 seconds
	"granularity": 40
},
{
	"endpoint": "endpointURL",
	"granularity": 40,
	"alias": "S3",
	//optional field to indicate that this endpoint should only be checked
	//for content of a certain type
	//if not present, all content will be checked against this endpoint
	"contentType": "Image"
}
],
```

```
//identifier of the platform the app is running on
//should also contain the environment
"platform": "aws-prod",
```

```
//feeder-specific configuration
//for each feeder, we need a new struct, new field in AppConfig for it, and
//handling for the feeder in startAggregator()
"splunk-config": {
	"logFilePath": "/var/log/apps/pam.log"
}
```
